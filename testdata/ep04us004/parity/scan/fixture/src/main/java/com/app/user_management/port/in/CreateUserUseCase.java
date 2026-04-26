package com.app.user_management.port.in;

import com.app.user_management.domain.User;

public interface CreateUserUseCase {
    User execute(String name, String email);
}
