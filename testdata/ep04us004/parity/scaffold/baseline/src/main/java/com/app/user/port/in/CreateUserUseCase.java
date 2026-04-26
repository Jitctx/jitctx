package com.app.user.port.in;

public interface CreateUserUseCase {

    UserResponse execute(CreateUserCommand cmd);
}
