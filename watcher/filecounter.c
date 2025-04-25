#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/inotify.h>
#include <errno.h>
#include <string.h>
#include <dirent.h>
#include <sys/stat.h>
#include <signal.h>
#include <poll.h>


#define BUF_LEN (10 * (sizeof(struct inotify_event) + 255))

volatile sig_atomic_t continue_running = 1;
int counter = 0;

int count_files_in_directory(const char *dir_path) {
    DIR *dir = opendir(dir_path);
    if (dir == NULL) {
        perror("opendir error");
        return -1;
    }

    int count = 0;
    struct dirent *entry;
    struct stat entry_stat;
    while ((entry = readdir(dir)) != NULL) {

        char full_path[1024];
        snprintf(full_path, sizeof(full_path), "%s/%s", dir_path, entry->d_name);

        if (stat(full_path, &entry_stat) == -1) {
            perror("stat error");
            continue;
        }

        if (S_ISREG(entry_stat.st_mode)) {
            count++;
        }
    }

    closedir(dir);
    return count;
}

void handle_signal(int sig) {
    continue_running = 0;
}

void stop_watching() {
    continue_running = 0;
}

void start_watching(const char* path) {
    continue_running = 1;
    counter = count_files_in_directory(path);
    if (counter == -1) {
        perror("count_files_in_directory error");
        return;
    }

    const char *dir_to_watch = path;
    int inotify_fd = inotify_init1(IN_NONBLOCK);
    if (inotify_fd == -1) {
        perror("inotify_init error");
        return;
    }

    int watch_fd = inotify_add_watch(inotify_fd, dir_to_watch, IN_CREATE | IN_DELETE);
    if (watch_fd == -1) {
        perror("inotify_add_watch error");
        close(inotify_fd);
        return;
    }

//    signal(SIGINT, handle_signal);
//    signal(SIGTERM, handle_signal);

    char buffer[BUF_LEN];

    
    struct pollfd fds[1];
    fds[0].fd = inotify_fd;
    fds[0].events = POLLIN;

    while (continue_running) {
        int poll_num = poll(fds, 1, 500);
        if (poll_num == -1) {
            if (errno == EINTR) continue;
            perror("poll error");
            break;
        }

        if (poll_num == 0) {
            continue;
        }

        if (fds[0].revents & POLLIN) {
            ssize_t num_read = read(inotify_fd, buffer, BUF_LEN);
            if (num_read < 0 && errno != EAGAIN) {
                perror("read error");
                break;
            }

            for (char *ptr = buffer; ptr < buffer + num_read;) {
                struct inotify_event *event = (struct inotify_event *) ptr;
                if (event->mask & IN_CREATE) {
                    //printf("File created: %s\n", event->name);
                    counter++;
                }
                if (event->mask & IN_DELETE) {
                    //printf("File deleted: %s\n", event->name);
                    counter--;
                }
                ptr += sizeof(struct inotify_event) + event->len;
            }
        }
    }

    inotify_rm_watch(inotify_fd, watch_fd);
    close(inotify_fd);
    printf("There are %d files in the dir.\n", counter);
    return;
}


//int main(int argc, char *argv[]) {
//    if (argc < 2) {
//        fprintf(stderr, "Usage: %s <directory_to_watch>\n", argv[0]);
//        return EXIT_FAILURE;
//    }
//    start_watching(argv[1]);
//}

