load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "portforward",
    srcs = ["constants.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/k8s.io/apimachinery/pkg/util/portforward",
    importpath = "k8s.io/apimachinery/pkg/util/portforward",
    visibility = ["//visibility:public"],
)

alias(
    name = "go_default_library",
    actual = ":portforward",
    visibility = ["//visibility:public"],
)
