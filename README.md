
## gitoperations

The gitoperations module is a Golang library for interacting with git via the command line executable.
There are other golang approaches to git integration such as a pure Golang implementation
([go-git](https://pkg.go.dev/github.com/go-git/go-git/v5))
or a wrapper around a C library ([git2go](https://github.com/libgit2/git2go)).
Gitoperations takes the approach of driving the actual git command line executable.
One motivation for this approach is that
[git plumbing](https://git-scm.com/book/en/v2/Git-Internals-Plumbing-and-Porcelain)
is designed to be scriptable.

The functions provided have been built up organically based on needs of the author,
but are not exhaustive. The library is designed using an interface to support easy mocking in your
application unit tests.
