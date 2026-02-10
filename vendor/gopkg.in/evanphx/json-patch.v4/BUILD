load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "json-patch_v4",
    srcs = [
        "errors.go",
        "merge.go",
        "patch.go",
    ],
    importmap = "github.com/fasterci/rules_gitops/vendor/gopkg.in/evanphx/json-patch.v4",
    importpath = "gopkg.in/evanphx/json-patch.v4",
    visibility = ["//visibility:public"],
    deps = ["//vendor/github.com/pkg/errors"],
)

alias(
    name = "go_default_library",
    actual = ":json-patch_v4",
    visibility = ["//visibility:public"],
)
