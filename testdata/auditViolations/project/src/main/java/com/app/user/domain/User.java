package com.app.user.domain;

import org.springframework.stereotype.Component;

// Violation: domain/ file imports org.springframework.* — triggers forbidden_import (domain-leak)
public class User {
    private Long id;
    private String name;
    private String email;
}
