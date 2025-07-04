package main

import (
	
	"fmt"
	"html/template"
	"strings"

	"net/http"
	
	"time"

	"github.com/gorilla/csrf"
)

// ---------------------------------------------------------------------
// ROOMS
// ---------------------------------------------------------------------

func GetRoomByID(roomID int) (Room, error) {
	var room Room
	err := db.QueryRow(`
	    SELECT room_id, room_type, status, price, category
	    FROM   rooms WHERE room_id = $1`, roomID).
		Scan(&room.RoomID, &room.RoomType, &room.Status, &room.Price, &room.Category)
	return room, err
}

func AddRoom(r Room) error {
	_, err := db.Exec(`
	    INSERT INTO rooms (room_type, status, price, category)
	    VALUES ($1, $2, $3, $4)`,
		r.RoomType, r.Status, r.Price, r.Category)
	return err
}

func UpdateRoom(r Room) error {
	_, err := db.Exec(`
	    UPDATE rooms
	    SET    room_type=$1, status=$2, price=$3, category=$4
	    WHERE  room_id=$5`,
		r.RoomType, r.Status, r.Price, r.Category, r.RoomID)
	return err
}

func DeleteRoom(id int) error {
	_, err := db.Exec(`DELETE FROM rooms WHERE room_id=$1`, id)
	return err
}

// ---------------------------------------------------------------------
// GUESTS
// ---------------------------------------------------------------------

func GetGuestByID(id int) (Guest, error) {
	var g Guest
	err := db.QueryRow(`
	    SELECT guest_id, name, email, phone
	    FROM   guests WHERE guest_id = $1`, id).
		Scan(&g.GuestID, &g.Name, &g.Email, &g.Phone)
	return g, err
}

func AddGuest(g Guest) (int, error) {
	var id int
	err := db.QueryRow(`
	    INSERT INTO guests (name, email, phone)
	    VALUES ($1, $2, $3) RETURNING guest_id`,
		g.Name, g.Email, g.Phone).Scan(&id)
	return id, err
}

func UpdateGuest(g Guest) error {
	_, err := db.Exec(`
	    UPDATE guests
	    SET    name=$1, email=$2, phone=$3
	    WHERE  guest_id=$4`,
		g.Name, g.Email, g.Phone, g.GuestID)
	return err
}

func DeleteGuest(id int) error {
	// сначала удаляем бронирования, потом гостя
	if _, err := db.Exec(`DELETE FROM bookings WHERE guest_id=$1`, id); err != nil {
		return err
	}
	_, err := db.Exec(`DELETE FROM guests WHERE guest_id=$1`, id)
	return err
}

// ---------------------------------------------------------------------
// BOOKINGS
// ---------------------------------------------------------------------

func AddBooking(b Booking) error {
	_, err := db.Exec(`
	    INSERT INTO bookings (room_id, guest_id, check_in, check_out,
	                          total_amount, paid, status)
	    VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		b.RoomID, b.GuestID, b.CheckIn, b.CheckOut,
		b.TotalAmount, b.Paid, b.Status)
	return err
}

func CancelBooking(id int) error {
	_, err := db.Exec(`UPDATE bookings SET status='cancelled' WHERE booking_id=$1`, id)
	return err
}

func MarkBookingPaid(id int) error {
	_, err := db.Exec(`UPDATE bookings SET paid=true WHERE booking_id=$1`, id)
	return err
}

// ---------------------------------------------------------------------
// Получение справочных данных
// ---------------------------------------------------------------------

func GetGuests() ([]Guest, error) {
	rows, err := db.Query(`SELECT guest_id, name, email, phone FROM guests`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Guest
	for rows.Next() {
		var g Guest
		if err := rows.Scan(&g.GuestID, &g.Name, &g.Email, &g.Phone); err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, rows.Err()
}

func GetRooms() ([]Room, error) {
	rows, err := db.Query(`SELECT room_id, room_type, status, price, category FROM rooms`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Room
	for rows.Next() {
		var r Room
		if err := rows.Scan(&r.RoomID, &r.RoomType, &r.Status, &r.Price, &r.Category); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// ---------------------------------------------------------------------
// Пользователи (админка)
// ---------------------------------------------------------------------

func GetUserByEmail(email string) (User, error) {
	var u User
	err := db.QueryRow(`
	    SELECT user_id, email, password, role
	    FROM   users WHERE email=$1`, email).
		Scan(&u.UserID, &u.Email, &u.Password, &u.Role)
	return u, err
}

func GetAllUsers() ([]User, error) {
	rows, err := db.Query(`SELECT user_id, email, role FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.UserID, &u.Email, &u.Role); err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

// ---------------------------------------------------------------------
// Универсальный рендер шаблонов (новая сигнатура Variant-B)
// ---------------------------------------------------------------------

// renderTemplate рендерит страницу, обязательно через каркас base.html.
func renderTemplate(w http.ResponseWriter,
    r *http.Request,
    data interface{},
    files ...string) {

    // 1. дописываем csrfField, если в data - это map
    if m, ok := data.(map[string]interface{}); ok {
        if _, exists := m["csrfField"]; !exists {
            m["csrfField"] = csrf.TemplateField(r)
        }
    }

    // 2. убеждаемся, что base.html стоит ПЕРВЫМ в списке
    if len(files) == 0 || !strings.HasSuffix(files[0], "base.html") {
        // вставляем base.html в начало (пусть путь будет относительный
        // так же, как в остальных вызовах)
        files = append([]string{"templates/base.html"}, files...)
    }

    // 3. парсим
    funcMap := template.FuncMap{"now": time.Now}
    tmpl, err := template.New("").Funcs(funcMap).ParseFiles(files...)
    if err != nil {
        http.Error(w, "template parse error: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // 4. выполняем шаблон "base"
    if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
        http.Error(w, "template exec error: "+err.Error(), http.StatusInternalServerError)
    }
}


// ---------------------------------------------------------------------
// Вспомогательные выводы ошибок / форм
// ---------------------------------------------------------------------

func showLoginForm(w http.ResponseWriter, r *http.Request, msg string) {
	renderTemplate(w, r,
		map[string]interface{}{"Message": msg},
		"templates/base.html",
		"templates/login.html")
}

func loginForm(w http.ResponseWriter, r *http.Request, msg string) {
	showLoginForm(w, r, msg) // просто алиас, чтобы не править хендлер
}

// ---------------------------------------------------------------------
// Фильтр бронирований
// ---------------------------------------------------------------------

func GetAllBookingsFiltered(f BookingFilter) ([]BookingView, error) {
	query := `SELECT booking_id, guest_name, room_id, room_type,
	                 check_in, check_out, status, paid
	          FROM   booking_view WHERE 1=1`
	args := []any{}
	idx  := 1

	if !f.CheckIn.IsZero() {
		query += fmt.Sprintf(" AND check_in >= $%d", idx)
		args = append(args, f.CheckIn); idx++
	}
	if !f.CheckOut.IsZero() {
		query += fmt.Sprintf(" AND check_out <= $%d", idx)
		args = append(args, f.CheckOut); idx++
	}
	if f.RoomID != 0 {
		query += fmt.Sprintf(" AND room_id = $%d", idx)
		args = append(args, f.RoomID); idx++
	}
	if f.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, f.Status); idx++
	}

	rows, err := db.Query(query, args...)
	if err != nil { return nil, err }
	defer rows.Close()

	var list []BookingView
	for rows.Next() {
		var b BookingView
		if err := rows.Scan(&b.BookingID, &b.GuestName, &b.RoomID, &b.RoomType,
			&b.CheckIn, &b.CheckOut, &b.Status, &b.Paid); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}
