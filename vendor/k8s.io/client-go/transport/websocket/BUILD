load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "websocket",
    srcs = ["roundtripper.go"],
    importmap = "github.com/fasterci/rules_gitops/vendor/k8s.io/client-go/transport/websocket",
    importpath = "k8s.io/client-go/transport/websocket",
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/github.com/gorilla/websocket",
        "//vendor/k8s.io/apimachinery/pkg/api/errors",
        "//vendor/k8s.io/apimachinery/pkg/apis/meta/v1:meta",
        "//vendor/k8s.io/apimachinery/pkg/runtime",
        "//vendor/k8s.io/apimachinery/pkg/runtime/serializer",
        "//vendor/k8s.io/apimachinery/pkg/util/httpstream",
        "//vendor/k8s.io/apimachinery/pkg/util/httpstream/wsstream",
        "//vendor/k8s.io/apimachinery/pkg/util/net",
        "//vendor/k8s.io/client-go/rest",
        "//vendor/k8s.io/client-go/transport",
    ],
)

alias(
    name = "go_default_library",
    actual = ":websocket",
    visibility = ["//visibility:public"],
)
