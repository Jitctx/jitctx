package com.app.user.adapter.in.web;

import com.app.user.port.in.CreateUserUseCase;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class UserController {

    private final CreateUserUseCase createUserUseCase;

    public UserController(CreateUserUseCase createUserUseCase) {
        this.createUserUseCase = createUserUseCase;
    }


    @PostMapping("/users")
    public Object postUsers() {
        // TODO(jitctx): implement UserController.postUsers
        throw new UnsupportedOperationException("Not yet implemented");
    }
}
