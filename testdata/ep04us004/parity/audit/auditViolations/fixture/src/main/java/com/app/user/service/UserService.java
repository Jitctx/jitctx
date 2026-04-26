package com.app.user.service;

import com.app.user.domain.User;
import com.app.user.port.in.CreateUser;
import org.springframework.stereotype.Service;

// Violation: service has a field whose type ends with Jpa — triggers field_type_layer_violation (adapter-injection)
@Service
public class UserService implements CreateUser {
    private final UserRepositoryJpa userRepositoryJpa;

    public UserService(UserRepositoryJpa userRepositoryJpa) {
        this.userRepositoryJpa = userRepositoryJpa;
    }

    @Override
    public User execute(String name, String email) {
        return null;
    }
}
