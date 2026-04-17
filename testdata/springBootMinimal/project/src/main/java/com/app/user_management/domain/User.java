package com.app.user_management.domain;

import jakarta.persistence.Entity;

@Entity
public class User {
    private Long id;
    private String name;
    private String email;
}
