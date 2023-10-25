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

## common issues

### stuck with patched deployment
in case your deployment is styck in patched state, use
```
gograpple rollback [namespace] [deployment]
```

### vscode
 > The debug session doesnt start until the entrypoint is triggered more than once.

 Review and remove any extra breakpoints you may have enabled, that you dont need (Run and Debug > Breakpoints panel). Vscode seems to like them saved across projects.
