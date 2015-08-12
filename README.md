# fswatcher

Simple fswatcher.   
Executes command when file or directories are modified.

## Usage

    fswatcher [options] [path...]

Options:

     --exec, -x     command to execute
     --include, -i  filter to include. e.g. .(go|rb|java)
     --exclude, -e  exclude paths matching REGEX.
     --log, -l      set log level
     --help, -h     show help
     --version, -v  print the version

Example:

    fswatcher --exec 'git diff' --include .go$ ./

You can use short option.

    fswatcher -x 'git diff' -i .go$ ./

## Version

0.0.1

## License

MIT

## Author

Shun Sugai