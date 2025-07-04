package main

import "time"

type Room struct {
    RoomID   int
    RoomType string
    Status   string
    Price    float64
    Category string
}




type Guest struct {
    GuestID int
    Name    string
    Email   string
    Phone   string
}



type Booking struct {
    BookingID   int
    RoomID      int
    GuestID     int
    CheckIn     time.Time
    CheckOut    time.Time
    TotalAmount float64    // <<< добавить это поле!
    Paid        bool
    Status      string
}




type BookingView struct {
    BookingID  int
    GuestName  string
    RoomID     int
    RoomType   string
    CheckIn    time.Time
    CheckOut   time.Time
    Status     string
    Paid       bool
}

type BookingFilter struct {
    CheckIn  time.Time
    CheckOut time.Time
    RoomID   int
    Status   string
}



type BookingsPageData struct {
    Bookings []BookingView
    Filter   BookingFilter
}


type RoomsPageData struct {
    Rooms []Room
}

type User struct {
    UserID   int
    Email    string
    Password string
    Role     string
}
