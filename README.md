It is just a learning project. It builds k8s operator from scratch. It does nothing fancy, just watches the defined path and writes count of files in it, when the resource is deleted.
Counting is done in C using inotify API and called over CGO directly from operator.
