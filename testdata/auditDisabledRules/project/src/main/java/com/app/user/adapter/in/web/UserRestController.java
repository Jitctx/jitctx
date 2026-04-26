package com.app.user.adapter.in.web;

import com.app.user.domain.User;
import org.springframework.web.bind.annotation.RestController;
import javax.persistence.Entity;

// Violation: @Entity is present but this file is NOT under domain/ — triggers entity-path-mismatch.
// (A realistic fixture would not combine @Entity and @RestController; this is intentionally
// synthetic to produce a second testable audit violation alongside domain-leak in User.java.)
@Entity
@RestController
public class UserRestController {
    public User getUser(Long id) {
        return null;
    }
}
