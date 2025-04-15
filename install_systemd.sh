#!/bin/bash

INSTALLDIR=/opt/tgbot/
BINARY=goWebTgMonitor-1.1.0_linux_amd64

mkdir $INSTALLDIR
cp $BINARY $INSTALLDIR
cp config.json $INSTALLDIR
cd $INSTALLDIR
ln -s $BINARY goWebTgMonitor
cd -

cp tgbot.service /etc/systemd/system/tgbot.service
systemctl daemon-reload

#sudo systemctl enable tgbot.service
#sudo systemctl start tgbot.service
systemctl status tgbot.service
#journalctl -u tgbot.service -f

