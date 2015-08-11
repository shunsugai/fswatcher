# fswatcher

Simple fswatcher.   
Executes command when file or directories are modified.

## Usage

    fswatcher --exec '<command to execute>' --include=REGEX <path>...

Example:

    fswatcher --exec 'git diff' --include .go$ ./

You can use short option.

    fswatcher -x 'git diff' -i .go$ ./

## License

MIT

## Author

Shun Sugai