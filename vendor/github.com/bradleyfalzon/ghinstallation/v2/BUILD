load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "ghinstallation",
    srcs = [
        "appsTransport.go",
        "sign.go",
        "transport.go",
    ],
    importmap = "github.com/fasterci/rules_gitops/vendor/github.com/bradleyfalzon/ghinstallation/v2",
    importpath = "github.com/bradleyfalzon/ghinstallation/v2",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/github.com/golang-jwt/jwt/v4:jwt",
        "//vendor/github.com/google/go-github/v68/github",
    ],
)

alias(
    name = "go_default_library",
    actual = ":ghinstallation",
    visibility = ["//visibility:public"],
)
