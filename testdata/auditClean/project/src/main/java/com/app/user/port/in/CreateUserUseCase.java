package com.app.user.port.in;

import com.app.user.domain.User;

public interface CreateUserUseCase {
    User execute(String name, String email);
}
