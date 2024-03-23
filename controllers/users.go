package controllers

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"

	// "golang.org/x/crypto/bcrypt"

	"github.com/jamesatomc/go-api/db"
	"github.com/jamesatomc/go-api/models"
	"gorm.io/gorm"
)

// Argon2 Hashing Function
func hashPassword(password string) (string, error) {
    hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
    if err != nil {
        return "", err
    }
    return hash, nil 
}

// Argon2 Password Comparison Function
func comparePassword(hashedPassword, password string) bool {
    match, err := argon2id.ComparePasswordAndHash(password, hashedPassword)
    return err == nil && match 
}

// FindUsers function
func FindUsers(c *gin.Context) {
	var users []models.User
	connect.Database.Find(&users)

	c.JSON(http.StatusOK, gin.H{"data": users})
}

// CreateUser function
func CreateUser(c *gin.Context) {
	// Validate input
	var input models.CreateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

    hashedPassword, err := hashPassword(input.Password)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
        return
    }
    
	user := models.User{
		Username: input.Username,
		Email:   input.Email,
		Password: hashedPassword,
		FirstName: input.FirstName,
		LastName:  input.LastName,
	}

    // Check for duplicate username
    var existingUser models.User
    if err := connect.Database.Where("username = ?", input.Username).First(&existingUser).Error; err != nil {
        if err != gorm.ErrRecordNotFound {
            // Handle other database errors
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking for username"})
            return
        }
        // Record not found - username is available
    } else {
        // Username already exists
        c.JSON(http.StatusConflict, gin.H{"error": "User"})
        return
    }
    // Use result instead of directly saving
    result := connect.Database.Create(&user)
        // Error handling:
    if result.Error != nil {
        // Check if the error is due to a duplicate email
        if strings.Contains(result.Error.Error(), "duplicate key value violates unique constraint") {
            c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating user"}) 
        }
        return
        }

        c.JSON(http.StatusOK, gin.H{"data": user})
}

// FindUser function
func FindUser(c *gin.Context) {
	var user models.User

	if err := connect.Database.Where("id = ?", c.Param("id")).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding user"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user})
}

// UpdateUser function
func UpdateUser(c *gin.Context) {
    // Get model if exist
    var user models.User
    if err := connect.Database.Where("id = ?", c.Param("id")).First(&user).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
        c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
        } else {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding user"})
        }
        return
    }

    // Validate input
    var input models.UpdateUserInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Check for duplicate email (if the email is being changed)
    if input.Email != user.Email {
        if err := connect.Database.Where("email = ?", input.Email).First(&models.User{}).Error; err != nil {
        if err != gorm.ErrRecordNotFound {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking for email"})
            return
        }
        } else {
        // Email already exists
        c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
        return
        }
    }

    // Check for duplicate username (if the username is being changed)
    if input.Username != user.Username {
        var existingUser models.User
        if err := connect.Database.Where("username = ?", input.Username).First(&existingUser).Error; err != nil {
            if err != gorm.ErrRecordNotFound { 
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking for username"})
                return
            } 
            // else -> Record not found, so username is available
        } else {
            // Username already exists 
            c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
            return
        }
    }

    // Update the user record
    if err := connect.Database.Model(&user).Updates(input).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating user"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"data": user})
}

//Login function
func Login(c *gin.Context) {
    var input models.User // Use a specific struct for login credentials
  
    if err := c.ShouldBindJSON(&input); err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }
  
    var user models.User
    if err := connect.Database.Where("username = ?", input.Username).First(&user).Error; err != nil {
      if err == gorm.ErrRecordNotFound {
        // Avoid revealing if it's username or password issue
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
        return
      }
      // Handle other potential database errors
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Error logging in"})
      return
    }
  
    // Compare hashed password
    if !comparePassword(user.Password, input.Password) {
      c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
      return
    }
  
    // Generate authentication token (consider using JWT)
    token, err := GenerateToken(user.ID)
    if err != nil {
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating token"})
      return
    }
  
    c.JSON(http.StatusOK, gin.H{"token": token})
  }
  
  func GenerateToken(userID uint) (string, error) {
    // Move secret key retrieval and storage outside the function (refer to previous improvements)
    secretKey := os.Getenv("JWT_SECRET_KEY")
  
    token := jwt.New(jwt.SigningMethodHS256)
  
    claims := token.Claims.(jwt.MapClaims)
    claims["user_id"] = userID
    claims["exp"] = time.Now().Add(time.Hour * 24).Unix() // Example expiration
  
    tokenString, err := token.SignedString([]byte(secretKey))
    if err != nil {
      return "", err
    }
  
    return tokenString, nil
}

func DeleteUser(c *gin.Context) {
    // Get the username from the request (route parameters, query, etc.)
    username := c.Param("username")  // Example: Assuming username in route parameter

    // Find the user to delete
    var user models.User
    if err := connect.Database.Where("username = ?", username).First(&user).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting user"})
        }
        return
    }

    // Delete the user
    if err := connect.Database.Delete(&user).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting user"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func UpdatePassword(c *gin.Context) {
    var input models.UpdatePasswordInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 1. Find the user by username
    var user models.User
    if err := connect.Database.Where("username = ?", input.Username).First(&user).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding user"})
        }
        return
    }

    // 2. Verify old password
    if !comparePassword(user.Password, input.OldPassword) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect old password"})
        return
    }

    // 3. Hash the new password
    newPasswordHashed, err := hashPassword(input.NewPassword) 
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing new password"})
        return
    }

    // 4. Check if the new password is already used
    for _, oldHash := range user.PasswordHistory {
        if comparePassword(oldHash, input.NewPassword) {
            c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot reuse recent passwords"})
            return
        }
    }

    // 5. Update the user's password and history
    connect.Database.Model(&user).Updates(map[string]interface{}{
        "password": newPasswordHashed,
        "password_history": append(user.PasswordHistory, newPasswordHashed), // Add new hash to history
    })

    // Limit password history (optional)
    if len(user.PasswordHistory) > 5 { // Store only last 5 passwords
        user.PasswordHistory = user.PasswordHistory[1:] // Remove oldest
        connect.Database.Save(&user) 
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}