# README

Instructions to debug the example app.

## STEPS

Install dependencies: 

    $ brew install helm kubectx k9s

Start docker, check which context you are connected to:

    $ kubectx
    docker-desktop

Setup local cluster, deploy test chart:

    $ cd test/app
    $ make build && make deploy

Use k9s to check if the example app pod is running:

    $ k9s

then patch:

    $ gograpple patch example --image golang

and debug (using vscode):
    
    $ gograpple delve example --source . --vscode

set a breakpoint in the HTTP handler.

> Note: before the startup of the service debugging does not work as delve is not started yet!

Use port forwarding via k9s to expose the pod (shift-F):

- App Port: 80
- Local Port: 8080

> Note: This will be reverted once you exit k9s!

Now access the web service by visiting: http://localhost:8080

You should be dropped into the debugger in VSCode at the breakpoint!

Once done, rollback:

    $ gograpple patch example --rollback

Happy debugging!