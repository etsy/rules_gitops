load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "websocket",
    srcs = [
        "client.go",
        "dial.go",
        "hybi.go",
        "server.go",
        "websocket.go",
    ],
    importmap = "github.com/fasterci/rules_gitops/vendor/golang.org/x/net/websocket",
    importpath = "golang.org/x/net/websocket",
    visibility = ["//visibility:public"],
)

alias(
    name = "go_default_library",
    actual = ":websocket",
    visibility = ["//visibility:public"],
)
