# What?

`amt` is an utility that makes a local mirror of Arch Linux packages.

# Why?

Arch Linux repository consists of two parts: the packages database and packages themself.
There are race conditions when packages are missing as files but still present in the database as records.
rsync is not the answer because it knows nothing about the database records.
So I created the `amt` to keep packages and their database in the guaranteed consistent state.
`amt` fetches the database, unpack it into memory and fetch packages according to database records.

# Build

Go >= 1.21 is required.

# Usage

See `./amt -h` for the details.
Config example:

```toml
[mirror.de]
enabled = false
arch = 'aarch64'
uri = 'https://mirrors.dotsrc.org/archlinuxarm/%arch%/%section%'
sections = ['core', 'extra', 'community', 'alarm']

[mirror.ge]
enabled = true
arch = 'x86_64'
uri = 'https://arch.grena.ge/%section%/os/%arch%'
sections = ['core', 'extra', 'community', 'multilib']
```

# License

GPL.
