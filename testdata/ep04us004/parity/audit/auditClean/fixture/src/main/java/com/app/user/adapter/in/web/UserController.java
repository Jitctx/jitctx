package com.app.user.adapter.in.web;

import com.app.user.port.in.CreateUserUseCase;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class UserController {
    private final CreateUserUseCase createUserUseCase;

    public UserController(CreateUserUseCase createUserUseCase) {
        this.createUserUseCase = createUserUseCase;
    }
}
