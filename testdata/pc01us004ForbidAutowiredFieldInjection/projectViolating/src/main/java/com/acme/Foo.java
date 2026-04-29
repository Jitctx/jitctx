package com.acme;

import org.springframework.beans.factory.annotation.Autowired;

public class Foo {

    @Autowired
    private UserRepo repo;
}

class UserRepo {}
