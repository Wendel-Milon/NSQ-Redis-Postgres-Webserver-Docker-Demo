package main

import "errors"

var ErrNoUserID = errors.New("no Userid provided in the Form")
var ErrNoPassWd = errors.New("no password provided in the Form")
