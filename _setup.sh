#!/bin/bash

set -e

command_exists () {
	command -v "$1" >/dev/null 2>&1 ;
}

if command_exists deliver ; then
	echo " (i) deliver already installed"
	exit 0
else
	echo " (i) deliver NOT yet installed, attempting install..."
fi

if command_exists rvm ; then
	echo " -> installing with RVM"
	gem install deliver
elif command_exists rbenv ; then
	echo " -> installing with rbenv"
	gem install deliver
	rbenv rehash
fi