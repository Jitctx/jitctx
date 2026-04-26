package com.app.user;

import com.app.user.port.in.CreateUser;
import org.springframework.web.bind.annotation.RestController;

// Violation: @RestController outside adapter/in/web/ — triggers annotation_path_mismatch (rest-controller-path-mismatch)
@RestController
public class UserController {
    private final CreateUser createUser;

    public UserController(CreateUser createUser) {
        this.createUser = createUser;
    }
}
