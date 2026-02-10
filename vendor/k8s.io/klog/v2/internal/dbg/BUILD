load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "dbg",
    srcs = ["dbg.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/k8s.io/klog/v2/internal/dbg",
    importpath = "k8s.io/klog/v2/internal/dbg",
    visibility = ["//vendor/k8s.io/klog/v2:__subpackages__"],
)

alias(
    name = "go_default_library",
    actual = ":dbg",
    visibility = ["//vendor/k8s.io/klog/v2:__subpackages__"],
)
