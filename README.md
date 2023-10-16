# 代理节点

## 安装 pcap

> 代理节点用到了 BPF 编译功能，需要依赖 `libpcap` 开发包。
> 
> 注意：是**_开发包_**。

### linux

```shell
# 不同 linux 的软件管理对该包有不同的名字，一般为 libpcap-devel 或 libpcap-dev，
# 一个名字搜不到可以试试另一个。

# ubuntu/debian 系
apt install libpcap-dev

# arch/manjaro 系
pacman -S libpcap-devel

# redhat/centos 系
yum install libpcap-devel
```

### windows

安装 [winpcap](https://www.winpcap.org/install/default.htm) 即可。

### darwin

未曾在苹果系统下开发部署过该程序，个人猜测未经验证：可能需要使用 [Homebrew](https://brew.sh/)
安装 [libpcap](https://formulae.brew.sh/formula/libpcap) 相关依赖。
