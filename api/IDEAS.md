~~config yaml for fvs to poll an endpoint to get videos~~

fvs stores videos in sqlite to remember between sessions perhaps?


api has endpoint to provide all files by host, that way the other client can check if any videos are missing

files can only be deleted if the other client has acknowledged the delete action
