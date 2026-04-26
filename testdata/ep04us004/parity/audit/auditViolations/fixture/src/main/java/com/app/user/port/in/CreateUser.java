package com.app.user.port.in;

import com.app.user.domain.User;

// Violation: interface in port/in/ does not end with UseCase — triggers interface_naming (port-naming)
public interface CreateUser {
    User execute(String name, String email);
}
