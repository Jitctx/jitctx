package com.acme.testsupport;

import org.springframework.beans.factory.annotation.Autowired;

public class Helper {

    @Autowired
    private UserRepo repo;
}

class UserRepo {}
