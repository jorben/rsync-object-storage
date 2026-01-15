<div align="center">
<h1>Rsync Object Storage</h1>

[中文版](README_zh.md)

[![Build]][build_url]
[![Version]][tag_url]
[![Size]][hub_url]
[![Pulls]][hub_url]
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg)](http://copyfree.org)
<p>A synchronization tool that monitors local file changes and syncs them to remote (S3) object storage in real-time.</p>

![rsync-object-storage](https://socialify.git.ci/jorben/rsync-object-storage/image?description=1&forks=1&issues=1&language=1&name=1&owner=1&pulls=1&stargazers=1&theme=Light)
</div>

### Features

- **Real-time Synchronization**
  - Monitors local file changes (including all subdirectories) and triggers real-time synchronization to remote object storage.
  - Deletions on the local filesystem are also synced to the remote storage (if you want to keep remote files, consider enabling versioning on your object storage bucket).
  - Supports hot file cooling: Files that trigger changes frequently within a configured window will only be synced once after the delay (configured via `sync.real_time.hot_delay`).
- **Scheduled Synchronization (Check Job)**
  - Compares all local files with their remote counterparts and syncs any differences (only syncs local files to remote; files that exist remotely but not locally will not be deleted).
  - Supports configurable task start time (`sync.check_job.start_at`), useful for scheduling full syncs during off-peak hours.
  - Supports configurable execution frequency (`sync.check_job.interval`), running periodically from the `start_at` time.
- **Flexible Modes**: Enable real-time sync, scheduled sync, or both independently.
- **Ignore Rules**: Support for ignoring files/directories based on name patterns, including `*` wildcards.

### Usage

#### Docker Compose

- **config.yaml** Example:

```yaml
# Local path configuration
local:
  path: /data # Local path to sync. In a container, this is the mapped path.

# Remote object storage configuration
remote:
  endpoint: cos.ap-guangzhou.myqcloud.com # e.g., cos.ap-guangzhou.myqcloud.com
  use_ssl: true
  secret_id: ${MY_SECRET_ID} # Can be set in environment variables
  secret_key: ${MY_SECRET_KEY} # Can be set in environment variables
  bucket: somebucket # Your bucket name
  region: ap-guangzhou # Your bucket region code (leave empty if not applicable)
  path: / # Remote path, use / for bucket root

# Sync configuration
sync:
  # Real-time sync configuration
  real_time:
    enable: true 
    hot_delay: 5 # Time in minutes to delay sync for frequently changed files (reduces API calls/bandwidth)
  
  # Scheduled check job configuration
  check_job:
    enable: true  # Enable periodic full scanning and sync
    interval: 24 # Frequency in hours
    start_at: 3:00:00 # Scheduled start time (recommended during low traffic)
  
  # Symlink handling strategy (skip|addr|file), default is skip
  # - skip: Ignore symbolic links
  # - addr: Save the link target address as a file in object storage
  # - file: Copy the actual file the link points to (uses addr for directories to avoid recursion)
  symlink: addr
  
  # Files and directories to ignore
  ignore:
    - .*.swp
    - "*~"
    - .DS_Store
    - .ds_store
    - .svn
    - .git
    - Thumbs.db
    - .idea

log:
  - writer: console
    formatter: console
    level: INFO 
  - writer: file
    formatter: json
    level: DEBUG
    format_config:
      time_fmt: "2006-01-02 15:04:05.000"
    write_config:
      log_path: "./log/ros_cos.log"
      max_size: 100
      max_age: 30
      compress: true
```

- **docker-compose.yml** Example:

```yaml
version: "3.8"

services:
  ros:
    container_name: ros
    image: jorbenzhu/rsync-object-storage:latest
    command: ["/app/ros", "-c", "/app/config.yaml"]
    environment:
      MY_SECRET_ID: # Your SECRET_ID
      MY_SECRET_KEY: # Your SECRET_KEY
    volumes:
      - /etc/localtime:/etc/localtime:ro
      - /data/ros/config.yaml:/app/config.yaml:ro # Your local config path
      - /data/sync_dir:/data:ro # The local directory you want to sync
    restart: always
```

### Troubleshooting

**"too many open files" or "no space left on device" errors**

This usually happens when the number of monitored directories exceeds the system limit. Adjusting the following settings can resolve this (for Docker, adjust these on the host machine):

- **Linux**: `/proc/sys/fs/inotify/max_user_watches`
- **BSD / OSX**: `kern.maxfiles` and `kern.maxfilesperproc`

```shell
# Linux:
# Increase the limit (adjust value as needed)
sudo sysctl fs.inotify.max_user_watches=102400 | sudo tee -a /etc/sysctl.conf
# Verify the change
cat /proc/sys/fs/inotify/max_user_watches
```

[build_url]: https://github.com/jorben/rsync-object-storage/
[hub_url]: https://hub.docker.com/r/jorbenzhu/rsync-object-storage/
[tag_url]: https://hub.docker.com/r/jorbenzhu/rsync-object-storage/tags

[Build]: https://github.com/jorben/rsync-object-storage/actions/workflows/dockerbuild.yml/badge.svg
[Size]: https://img.shields.io/docker/image-size/jorbenzhu/rsync-object-storage/latest?color=066da5&label=size
[Pulls]: https://img.shields.io/docker/pulls/jorbenzhu/rsync-object-storage.svg?style=flat&label=pulls&logo=docker
[Version]: https://img.shields.io/docker/v/jorbenzhu/rsync-object-storage/latest?arch=amd64&sort=semver&color=066da5
