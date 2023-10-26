# gograpple

gograpple that go program and delve into the high seas ...
or in other words: delve debugger injection for your golang code running in k8 pods

## requirements
 - helm
 - kubectl
 - docker

## quick start
```
brew install foomo/gograpple/gograpple
OR
go install github.com/foomo/gograpple@latest
```
start patch debugging in interactive mode
```
gograpple interactive
```
when you configure your patch correctly a file will be saved in your cwd and the debug session will start immmediatelly
## configuration
### patch (default)
| field | default value | description |
|---|---|---|
| source_path    |                | absolute path to the main.go (entrypoint) |
| cluster        |                | cluster context to use |
| namespace      |                | kubernetes namespace |
| deployment     |                | kubernetes deployment |
| container      |                | pod container to use |
| listen_addr    | 127.0.0.1:2345 | address to listen on for delve server |
| image          | alpine:latest  | image to use as base when building the patch |
| delve_continue | false          | continue the debugged process on start |
| launch_vscode  | false          | launch vscode with debug config |
### example config explained
if we use the following gograppe-patch example:
```
source_path: /home/runz0rd/dev/backend/cmd/search/search.go
cluster: gke_my-awesome-webshop-stage_europe-west1_default
namespace: stage-a
deployment: search-service-default
container: search
listen_addr: 127.0.0.1:2345
source_path: alpine:latest
delve_continue: false
launch_vscode: true
```
the following will happen:
 - your application at specified `source_path` will be built with base image `image` into a patch image
 - that patch image will be pushed into the same repo as the image thats originally deployed, for example `my-image-repo.com/backend/search-service:some-tag` will be `my-image-repo.com/backend/search-service-patch:latest`
 - the `deployment` you specified in `namespace` and `cluster` will be patched to allow running a delve server on it with your application
 - delve server will be started in your `container` and port-forwarded to be on `listen_addr`
 - if configured `delve_continue` will be applied on dlv startup and `launch_vscode` will simplify the debug session for vscode users

## common issues

### stuck with patched deployment
in case your deployment is styck in patched state, use
```
gograpple rollback [namespace] [deployment]
```

### vscode
 > The debug session doesnt start until the entrypoint is triggered more than once.

 Review and remove any extra breakpoints you may have enabled, that you dont need (Run and Debug > Breakpoints panel). Vscode seems to like them saved across projects.
