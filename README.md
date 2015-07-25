# mihari

Simple fswatcher.   
Executes command when file or directories are modified.

## Usage

    mihari --exec '<command to execute>' <path>... 

Example:

    mihari --exec 'git diff' ./

You can use short option.

    mihari -e 'git diff' ./

## License

MIT

## Author

Shun Sugai