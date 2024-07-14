pkgname=amt
pkgver=$(date +'%Y%m%d%H%M%S')
pkgrel=1
pkgdesc='Utility that makes a local mirror of Arch Linux packages.'
arch=('x86_64')
url='https://github.com/vimusov/amt'
license=('GPL')
makedepends=('go' 'just')
source=()

package()
{
    cd "$srcdir"
    just install "$pkgdir"
}
