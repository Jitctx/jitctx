package com.acme;

import org.springframework.beans.factory.annotation.Autowired;

public class Foo {

    private final UserRepo repo;

    public Foo(@Autowired UserRepo repo) {
        this.repo = repo;
    }
}

class UserRepo {}
