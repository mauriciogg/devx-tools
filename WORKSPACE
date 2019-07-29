workspace(name = "devx")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "new_git_repository")

# Maven rule for transitive dependencies
RULES_JVM_EXTERNAL_TAG = "1.2"
RULES_JVM_EXTERNAL_SHA = "e5c68b87f750309a79f59c2b69ead5c3221ffa54ff9496306937bfa1c9c8c86b"

http_archive(
    name = "rules_jvm_external",
    strip_prefix = "rules_jvm_external-%s" % RULES_JVM_EXTERNAL_TAG,
    sha256 = RULES_JVM_EXTERNAL_SHA,
    url = "https://github.com/bazelbuild/rules_jvm_external/archive/%s.zip" % RULES_JVM_EXTERNAL_TAG,
)

load("@rules_jvm_external//:defs.bzl", "maven_install")

# gRPC Java
# Note that this needs to come before io_bazel_go_rules. Both depend on
# protobuf and the version that io_bazel_rules_go depends on is broken for
# java, so io_grpc_grpc_java needs to get the dep first.
http_archive(
    name = "io_grpc_grpc_java",
    sha256 = "9bc289e861c6118623fcb931044d843183c31d0e4d53fc43c4a32b56d6bb87fa",
    strip_prefix = "grpc-java-1.21.0",
    urls = ["https://github.com/grpc/grpc-java/archive/v1.21.0.tar.gz"],
)


load("@io_grpc_grpc_java//:repositories.bzl", "grpc_java_repositories")

grpc_java_repositories()

# Go toolchains
http_archive(
    name = "io_bazel_rules_go",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.18.4/rules_go-0.18.4.tar.gz",
    sha256 = "3743a20704efc319070957c45e24ae4626a05ba4b1d6a8961e87520296f1b676",
)

load(
    "@io_bazel_rules_go//go:deps.bzl",
    "go_rules_dependencies",
    "go_register_toolchains",
)

go_rules_dependencies()
go_register_toolchains()

http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.17.0/bazel-gazelle-0.17.0.tar.gz"],
    sha256 = "3c681998538231a2d24d0c07ed5a7658cb72bfb5fd4bf9911157c0e9ac6a2687",
)

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
gazelle_dependencies()

go_repository(
    name = "org_golang_x_sync",
    commit = "1d60e4601c6fd243af51cc01ddf169918a5407ca",
    importpath = "golang.org/x/sync",
)

new_git_repository(
    name = "com_github_google_gousb",
    remote = "https://github.com/google/gousb.git",
    commit = "64d82086770b8b671e1e7f162372dd37f1f5efba",
    # Use custom BUILD file since we need to specify how to link agains libusb.
    build_file = "@//:BUILD.gousb",
)

maven_install(
    artifacts = [
        "androidx.annotation:annotation:1.1.0",
        "androidx.core:core:1.0.2",
        "androidx.test:monitor:1.2.0",
        "androidx.test:rules:1.2.0",
        "androidx.test:runner:1.2.0",
        "com.android.support:support-annotations:28.0.0",
        "com.android.support.test:runner:1.0.2",
        "com.google.code.findbugs:jsr305:3.0.2",
        "com.google.dagger:dagger:2.23.2",
        "com.google.dagger:dagger-compiler:2.23.2",
        "io.grpc:grpc-all:1.16.1",
        "io.grpc:grpc-testing:1.16.1",
        "javax.inject:javax.inject:1",
        "junit:junit:4.12",
        "org.apache.commons:commons-compress:1.10",
        "org.junit.jupiter:junit-jupiter-engine:5.3.2",
        "org.mockito:mockito-core:2.28.2",
        "org.mockito:mockito-android:2.28.2",
    ],
    repositories = [
        "https://maven.google.com",
        "https://repo1.maven.org/maven2",
        "https://mvnrepository.com",
    ],
)


# Android libs
android_sdk_repository(name = "androidsdk")

http_archive(
    name = "android_test_support",
    strip_prefix = "android-test-androidx-test-1.2.0",
    urls = ["https://github.com/android/android-test/archive/androidx-test-1.2.0.tar.gz"],
    sha256 = "01a3a6a88588794b997b46a823157aea06be9bcdc41800b61199893121ef26a3",
)

load("@android_test_support//:repo.bzl", "android_test_repositories")
android_test_repositories()
