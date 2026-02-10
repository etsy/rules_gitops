load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "v1beta1",
    srcs = [
        "certificatesigningrequest.go",
        "expansion_generated.go",
    ],
    importmap = "github.com/fasterci/rules_gitops/vendor/k8s.io/client-go/listers/certificates/v1beta1",
    importpath = "k8s.io/client-go/listers/certificates/v1beta1",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/k8s.io/api/certificates/v1beta1",
        "//vendor/k8s.io/apimachinery/pkg/labels",
        "//vendor/k8s.io/client-go/listers",
        "//vendor/k8s.io/client-go/tools/cache",
    ],
)

alias(
    name = "go_default_library",
    actual = ":v1beta1",
    visibility = ["//visibility:public"],
)
