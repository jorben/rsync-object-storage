# rsync-object-storage
一个同步工具，可以监听本地文件变更，同步到远端（s3）对象存储



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