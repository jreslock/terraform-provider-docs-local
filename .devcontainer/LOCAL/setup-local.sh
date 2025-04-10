#!/usr/bin/env bash

# Clone my dotfiles repo and run the setup script to install all the things

git clone https://github.com/jreslock/dotfiles.git $HOME/dotfiles
$HOME/dotfiles/setup
