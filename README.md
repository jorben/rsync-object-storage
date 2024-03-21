# rsync-object-storage
一个同步工具，可以监听本地文件变更，实时同步到远端（s3）对象存储

[![Build]][build_url]
[![Version]][tag_url]
[![Size]][hub_url]
[![Pulls]][hub_url]
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg)](http://copyfree.org)

### 功能点

- 实时同步
  - 支持监听本地路径下（含所有子目录）文件变更事件，实时发起同步本地变更到远端对象存储
  - 本地的删除操作也会同步删除远端对应文件（若不想删除远端建议通过启用对象存储的版本控制来实现）
  - 支持热点文件降温，配置时间内反复触发变更的文件，降低同步频率，（配置文件中`sync.real_time.hot_delay`配置项）
- 定时同步
  - 支持比对本地路径下全部文件与远端对应文件的差异，对存在差异的文件进行同步（只针对本地存在的文件操作同步，本地不存在但远端存在的文件不会被删除）
  - 支持指定首次任务启动时间点（配置文件中`sync.check_job.start_at`配置项），便于指定在非繁忙时点开始定期同步
  - 支持指定任务执行频率（配置文件中`sync.check_job.interval`配置项），将按周期在start_at时点启动
- 支持单独启用实时或定时同步（配置文件中`sync.real_time.enable`和`sycn.check_job.enable`配置项）
- 支持忽略，可按文件名/目录名称匹配，支持名称中含*通配（配置文件中`sync.ignore`配置项）

### 使用方法
#### Docker compose
- config.yaml 示例
```yaml
# 本地路径配置
local:
  path: /data # 本地需要同步的路径，容器中则为所需同步的路径映射容器中的路径

# 远端对象存储配置
remote:
  endpoint: cos.ap-guangzhou.myqcloud.com # 例如：cos.ap-guangzhou.myqcloud.com
  use_ssl: true
  secret_id: ${MY_SECRET_ID} # 可设置在环境变量中
  secret_key: ${MY_SECRET_KEY} # 可设置在环境变量中
  bucket: somebucket # 这里配置你的存储桶名称
  region: ap-guangzhou # 这里配置存储桶所在区域代码，无区域可留空
  path: / # 这里配置远端路径，桶根路径可配置为/

# 同步配置
sync:
  # 是否启用实时同步（监听本地文件变更进行同步，仅同步服务运行期间发生变更的文件，可结合check_job实现全量同步）
  real_time:
    enable: true 
    hot_delay: 5 # 在该时间内反复触发变更的热点文件将在该配置的时间内仅最后做1次同步动作，单位分钟（可有效减少反复变更带来的流量消耗）
  # 是否启用定期文件对账（扫描对比本地与远端文件差异进行同步）
  check_job:
    enable: true  # 是否启用定时全量检查和同步（检查存在差异时会触发差异文件的同步）
    interval: 24 # 文件对账频率间隔，单位小时
    start_at: 3:00:00 # 文件对账启动时间（建议选在凌晨），将结合频率间隔配置定期执行
  # symlink 由于对象存储不支持符号链接，所以需要选择对符号链接文件的处理策略，可选(skip|addr|file)，默认为skip
  # - skip 跳过符号链接文件，相当于忽略掉符号链接文件
  # - addr 把链接指向的地址保存到对象存储，用于记录链接的目标
  # - file 复制链接指向的实体文件到对象存储（为避免循环引用，符号链接指向文件夹时则会采用addr策略）
  symlink: addr
  # 忽略不同步的文件和文件夹
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
- docker-compose.yml 示例
```yaml
version: "3.8"

services:
  ros:
    container_name: ros
    image: jorbenzhu/rsync-object-storage:latest
    command: ["/app/ros", "-c", "/app/config.yaml"]
    environment:
      MY_SECRET_ID: #在这里配置你的对象存储SECRET_ID
      MY_SECRET_KEY: #在这里配置你的对象存储SECRET_KEY
    volumes:
      - /etc/localtime:/etc/localtime:ro
      - /data/ros/config.yaml:/app/config.yaml:ro #这里替换你的本地配置文件路径，映射为容器的/app/config.yaml路径
      - /data/sync_dir:/data:ro #这里替换你需要同步的本地路径，映射为容器的/data路径
    restart: always
```

### 注意事项

**遇到"too many open files"或"no space left on device"错误**

原因可能是需要监听的目录数量超出了系统的配置数量，调整到合理数值可解决（docker容器环境需要调整宿主机的配置）

- Linux: /proc/sys/fs/inotify/max_user_watches contains the limit, reaching this limit results in a “no space left on device” error.
- BSD / OSX: sysctl variables “kern.maxfiles” and “kern.maxfilesperproc”, reaching these limits results in a “too many open files” error.
```shell
# Linux：
# 调整系统配置，数值可以依据自身情况调整
sudo sysctl fs.inotify.max_user_watches=102400 | sudo tee -a /etc/sysctl.conf
# 查询是否生效
cat /proc/sys/fs/inotify/max_user_watches

```

[build_url]: https://github.com/jorben/rsync-object-storage/
[hub_url]: https://hub.docker.com/r/jorbenzhu/rsync-object-storage/
[tag_url]: https://hub.docker.com/r/jorbenzhu/rsync-object-storage/tags

[Build]: https://github.com/jorben/rsync-object-storage/actions/workflows/dockerbuild.yml/badge.svg
[Size]: https://img.shields.io/docker/image-size/jorbenzhu/rsync-object-storage/latest?color=066da5&label=size
[Pulls]: https://img.shields.io/docker/pulls/jorbenzhu/rsync-object-storage.svg?style=flat&label=pulls&logo=docker
[Version]: https://img.shields.io/docker/v/jorbenzhu/rsync-object-storage/latest?arch=amd64&sort=semver&color=066da5