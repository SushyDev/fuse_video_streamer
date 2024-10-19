package chart

import (
	"context"
	"debrid_drive/config"
	"fmt"

	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"

	// "github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/widgets/donut"
	"github.com/mum4k/termdash/widgets/linechart"
	"github.com/mum4k/termdash/widgets/text"
)

var ChartsOpen = 0

type LinechartData struct {
	BufferStartPosition int64
	SeekPosition        int64
	BufferLen           int64
	BufferCap           int64
}

func BytesToMegabytesRound(bytes int64) float64 {
	gb := float64(bytes) / 1024 / 1024

	return gb
}

func appendWithLimit(slice []float64, value float64, limit int) []float64 {
	slice = append(slice, value)
	if len(slice) > limit {
		slice = slice[1:]
	}
	return slice
}

type SeekTotal struct {
	SeekPosition int64
	TotalSize    int64
}

type Chart struct {
	StreamLogChannel chan string
	BufferLogChannel chan string
	ChartDataChannel chan LinechartData
	ChartStopChannel chan struct{}

	SeekTotal chan SeekTotal

	SeekPosition chan int64
}

func NewChart() *Chart {
	chart := &Chart{
		StreamLogChannel: make(chan string),
		BufferLogChannel: make(chan string),
		ChartDataChannel: make(chan LinechartData),
		ChartStopChannel: make(chan struct{}),

		SeekTotal: make(chan SeekTotal),
	}

	if config.Chart {
		go chart.Start()
	}

	return chart
}

func (chart *Chart) Start() {
	t, err := tcell.New()
	if err != nil {
		panic(err)
	}
	defer t.Close()

	ctx, cancel := context.WithCancel(context.Background())
	lc, err := linechart.New(
		linechart.AxesCellOpts(cell.FgColor(cell.ColorWhite)),
		linechart.YLabelCellOpts(cell.FgColor(cell.ColorWhite)),
		linechart.XLabelCellOpts(cell.FgColor(cell.ColorWhite)),
		linechart.YAxisAdaptive(),
	)
	if err != nil {
		panic(err)
	}

	bufferLog, err := text.New(text.RollContent(), text.WrapAtWords())
	if err != nil {
		panic(err)
	}

	streamLog, err := text.New(text.RollContent(), text.WrapAtWords())
	if err != nil {
		panic(err)
	}

	donutSeek, err := donut.New(
		donut.Label("Seek position"),
	)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case streamMessage := <-chart.StreamLogChannel:
				streamLog.Write(streamMessage)
			case bufferMessage := <-chart.BufferLogChannel:
				bufferLog.Write(bufferMessage)
			}
		}
	}()

	go func() {
		LineSnapshot := 128

		bufferStartPositions := []float64{}
		seekPositions := []float64{}
		BufferLens := []float64{}
		BufferCaps := []float64{}

		BufferCapOpts := []linechart.SeriesOption{
			linechart.SeriesCellOpts(cell.FgColor(cell.ColorWhite)),
		}

		BufferLenOpts := []linechart.SeriesOption{
			linechart.SeriesCellOpts(cell.FgColor(cell.ColorRed)),
		}

		seekPositionOpts := []linechart.SeriesOption{
			linechart.SeriesCellOpts(cell.FgColor(cell.ColorGreen)),
		}

		bufferStartPositionOpts := []linechart.SeriesOption{
			linechart.SeriesCellOpts(cell.FgColor(cell.ColorBlue)),
		}

		for {
			select {
			case <-ctx.Done():
				return
			case seekTotalData := <-chart.SeekTotal:
				if seekTotalData.SeekPosition == 0 && seekTotalData.TotalSize == 0 {
					continue
				}

				donutSeek.Absolute(int(seekTotalData.SeekPosition), int(seekTotalData.TotalSize))
			case chartData := <-chart.ChartDataChannel:
				if chartData.BufferStartPosition == 0 && chartData.SeekPosition == 0 && chartData.BufferLen == 0 {
					continue
				}

				BufferCaps = appendWithLimit(BufferCaps, BytesToMegabytesRound(chartData.BufferCap), LineSnapshot)
				bufferStartPositions = appendWithLimit(bufferStartPositions, BytesToMegabytesRound(chartData.BufferStartPosition), LineSnapshot)
				seekPositions = appendWithLimit(seekPositions, BytesToMegabytesRound(chartData.SeekPosition), LineSnapshot)
				BufferLens = appendWithLimit(BufferLens, BytesToMegabytesRound(chartData.BufferLen), LineSnapshot)

				if err := lc.Series("buffer_cap", BufferCaps, BufferCapOpts...); err != nil {
					panic(err)
				}

				if err := lc.Series("buffer_len", BufferLens, BufferLenOpts...); err != nil {
					panic(err)
				}

				if err := lc.Series("seek_position", seekPositions, seekPositionOpts...); err != nil {
					panic(err)
				}

				if err := lc.Series("buffer_start", bufferStartPositions, bufferStartPositionOpts...); err != nil {
					panic(err)
				}
			default:
			}
		}
	}()

	c, err := container.New(
		t,
		container.SplitVertical(
			container.Left(
				container.SplitHorizontal(
					container.Top(
						container.PlaceWidget(streamLog),
					),
					container.Bottom(
						container.PlaceWidget(bufferLog),
					),
				),
			),
			container.Right(
				container.SplitHorizontal(
					container.Top(
						container.PlaceWidget(lc),
					),
					container.Bottom(
						container.SplitVertical(
							container.Left(),
							container.Right(
								container.PlaceWidget(donutSeek),
							),
						),
					),
				),
			),
		),
	)
	if err != nil {
		panic(err)
	}

	if err := termdash.Run(ctx, t, c, termdash.RedrawInterval(250)); err != nil {
		panic(err)
	}

	ChartsOpen += 1
	defer func() {
		ChartsOpen -= 1
	}()

	defer c.Draw()

	for {
		select {
		case <-ctx.Done():
            cancel()
			return
		case <-chart.ChartStopChannel:
			cancel()
		default:
		}
	}
}

func (chart *Chart) Close() {
    if !config.Chart {
        return
    }

	select {
	case chart.ChartStopChannel <- struct{}{}:
	default:
	}
}

func (chart *Chart) Log(channel chan string, message string) {
    if !config.Chart {
        fmt.Printf("%s", message)
        return
    }

	select {
	case channel <- message:
	default:
	}
}

func (chart *Chart) LogStream(message string) {
	chart.Log(chart.StreamLogChannel, message)
}

func (chart *Chart) LogBuffer(message string) {
	chart.Log(chart.BufferLogChannel, message)
}

func (chart *Chart) UpdateSeekTotal(seekPosition int64, totalSize int64) {
    if !config.Chart {
        return
    }

	select {
	case chart.SeekTotal <- SeekTotal{
		SeekPosition: seekPosition,
		TotalSize:    totalSize,
	}:
	default:
	}
}
