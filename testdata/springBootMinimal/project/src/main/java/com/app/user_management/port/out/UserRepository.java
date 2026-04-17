package com.app.user_management.port.out;

import com.app.user_management.domain.User;
import java.util.Optional;

public interface UserRepository {
    Optional<User> findById(Long id);
    User save(User user);
}
