package com.app.user_management.domain;

@Entity
@Table(name = "users")
public class UserWithTableAnnotation {
    private Long id;
    private String email;
}
