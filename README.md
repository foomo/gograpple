# gograpple

gograpple that go program and delve into the high seas ...
or in other words: delve debugger injection for your golang code running in k8 pods

## common issues

### vscode
 > The debug session doesnt start until the entrypoint is triggered more than once.

 Review and remove any extra breakpoints you may have enabled, that you dont need (Run and Debug > Breakpoints panel). Vscode seems to like them saved across projects.
