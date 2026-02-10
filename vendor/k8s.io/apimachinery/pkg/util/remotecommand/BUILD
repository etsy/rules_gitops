load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "remotecommand",
    srcs = ["constants.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/k8s.io/apimachinery/pkg/util/remotecommand",
    importpath = "k8s.io/apimachinery/pkg/util/remotecommand",
    visibility = ["//visibility:public"],
    deps = ["//vendor/k8s.io/apimachinery/pkg/apis/meta/v1:meta"],
)

alias(
    name = "go_default_library",
    actual = ":remotecommand",
    visibility = ["//visibility:public"],
)
