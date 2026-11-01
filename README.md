<p align="center">
  <img src="resources/icon.png" width="96" alt="BITERNET logo" />
</p>

<h1 align="center">biternet-go</h1>

<p align="center">
  使用 Go 和 MyGo 编写的北京理工大学校园网客户端
</p>

<p align="center">
  <a href="https://github.com/bitoolkit/biternet-go/releases/latest">下载正式版</a>
  &nbsp;|&nbsp;
  <a href="https://pkg.go.dev/github.com/bitoolkit/biternet-go/client">Go 文档</a>
</p>

支持 Windows、macOS 和 Linux。

- 登录、注销与自动登录。
- 查看已用流量、在线时长、IP 地址和账户余额。
- 开机启动、托盘运行和启动时隐藏窗口。
- 浅色、深色和跟随系统外观。
- 新版本提醒。

## 使用

输入学号和统一认证密码，点击登录或按 Enter。

开启“自动登录”后，检测到离线时会尝试重新登录。手动注销或认证被拒绝后，自动登录暂停；再次手动登录或重新开启自动登录即可恢复。

“记住密码”会将凭据以明文保存在本机的用户配置目录 `biternet-go/settings.json` 中。取消勾选后，清除已保存的学号和密码。

关闭窗口会收起到托盘，通过托盘菜单可以重新打开或退出。系统不支持托盘时，关闭窗口直接退出。开机启动、启动时隐藏窗口和外观可在偏好设置中调整。

| 快捷键 | 操作 |
| --- | --- |
| Enter | 登录 |
| Ctrl+, | 打开偏好设置 |
| Esc | 返回主界面，或将主界面收起到托盘 |
| Ctrl+Q | 退出 |

macOS 使用 Command 代替 Ctrl。

## 运行与构建

需要 Go 1.27.3 或更新版本。Linux 还需要 GTK 3、Pango、Cairo 和 Fontconfig。

```sh
go run .
```

生成当前平台的安装包，产物位于 `build/`：

```sh
go run github.com/egoist/mygo/cmd/mygo build
```

安装后，使用 `--silent` 参数可启动到托盘。

## Nix

```sh
nix run github:bitoolkit/biternet-go
nix build github:bitoolkit/biternet-go
```

支持 Linux x86_64 和 aarch64。flake 同时提供 overlay、NixOS 和 Home Manager 模块。在自己的 flake 中添加：

```nix
inputs.biternet-go.url = "github:bitoolkit/biternet-go";
```

然后在 Home Manager 配置中启用：

```nix
imports = [ inputs.biternet-go.homeManagerModules.default ];
services.biternet = {
  enable = true;
  autoStart = true;
  silentStart = true;
};
```

NixOS 使用 `inputs.biternet-go.nixosModules.default`。

## Go client

`client/` 是独立的 Go 模块，可用于查询连接状态、登录和注销。仅依赖标准库，需要 Go 1.23 或更新版本。接口与示例见 [Go 文档](https://pkg.go.dev/github.com/bitoolkit/biternet-go/client)。

## 致谢

界面使用 [MyGo](https://github.com/egoist/mygo)，字体使用 Manrope 和 Noto Sans SC。字体许可见 [Manrope OFL](internal/desktop/fonts/Manrope-OFL.txt) 和 [Noto Sans SC OFL](internal/desktop/fonts/NotoSansSC-OFL.txt)。

## LICENSE

[Unlicense](LICENSE). Made by [BIToolkit](https://github.com/bitoolkit) with ♥️
