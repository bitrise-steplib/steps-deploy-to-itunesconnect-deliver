#!/bin/bash

command_exists () {
	command -v "$1" >/dev/null 2>&1 ;
}

gem_name="deliver"

if command_exists $gem_name ; then
	echo " (i) $gem_name already installed"
	exit 0
else
	echo " (i) $gem_name NOT yet installed, attempting install..."
fi

STARTTIME=$(date +%s)

which_ruby="$(which ruby)"
osx_system_ruby_pth="/usr/bin/ruby"
brew_ruby_pth="/usr/local/bin/ruby"

echo
echo " (i) Which ruby: $which_ruby"
echo " (i) Ruby version: $(ruby --version)"
echo

set -e

if [[ "$which_ruby" == "$osx_system_ruby_pth" ]] ; then
	echo " -> using system ruby - requires sudo"
	echo '$' sudo gem install ${gem_name} --no-document
	sudo gem install ${gem_name} --no-document
elif [[ "$which_ruby" == "$brew_ruby_pth" ]] ; then
	echo " -> using brew ($brew_ruby_pth) ruby"
	echo '$' gem install ${gem_name} --no-document
	gem install ${gem_name} --no-document
elif command_exists rvm ; then
	echo " -> installing with RVM"
	echo '$' gem install ${gem_name} --no-document
	gem install ${gem_name} --no-document
elif command_exists rbenv ; then
	echo " -> installing with rbenv"
	echo '$' gem install ${gem_name} --no-document
	gem install ${gem_name} --no-document
	echo '$' rbenv rehash
	rbenv rehash
else
	echo " [!] Failed to install: no ruby is available!"
	exit 1
fi

ENDTIME=$(date +%s)
echo
echo " (i) Setup took $(($ENDTIME - $STARTTIME)) seconds to complete"
echo
