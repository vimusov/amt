target := "amt"

default: build

build:
    cd src && go build -o ../{{target}}

clean:
    rm -f {{target}}

install destdir: build
    install -D -m 0755 {{target}} "{{destdir}}"/usr/bin/{{target}}
