load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "severity",
    srcs = ["severity.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/k8s.io/klog/v2/internal/severity",
    importpath = "k8s.io/klog/v2/internal/severity",
    visibility = ["//vendor/k8s.io/klog/v2:__subpackages__"],
)

alias(
    name = "go_default_library",
    actual = ":severity",
    visibility = ["//vendor/k8s.io/klog/v2:__subpackages__"],
)
