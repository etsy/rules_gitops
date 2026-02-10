load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "storagemigration",
    srcs = ["interface.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/k8s.io/client-go/informers/storagemigration",
    importpath = "k8s.io/client-go/informers/storagemigration",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/k8s.io/client-go/informers/internalinterfaces",
        "//vendor/k8s.io/client-go/informers/storagemigration/v1alpha1",
    ],
)

alias(
    name = "go_default_library",
    actual = ":storagemigration",
    visibility = ["//visibility:public"],
)
