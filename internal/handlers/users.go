package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"neosec/internal/db"
	"neosec/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetMe(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(primitive.ObjectID)

	var user models.User
	db.UsersCollection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)

	// Fetch Friends
	var friends []models.User
	if len(user.Friends) > 0 {
		cursor, _ := db.UsersCollection.Find(context.Background(), bson.M{"_id": bson.M{"$in": user.Friends}})
		cursor.All(context.Background(), &friends)
	}

	// Fetch Pending Incoming Requests
	var incomingReqs []models.FriendRequest
	incCursor, _ := db.FriendRequestsCollection.Find(context.Background(), bson.M{"to_user_id": userID, "status": "pending"})
	incCursor.All(context.Background(), &incomingReqs)

	// Fetch user details for incoming requests
	incomingHTML := ""
	if len(incomingReqs) == 0 {
		incomingHTML = `<div style="text-align:center; padding: 40px; color: #949ba4;">No pending friend requests.</div>`
	}
	for _, req := range incomingReqs {
		var fromUser models.User
		db.UsersCollection.FindOne(context.Background(), bson.M{"_id": req.FromUserID}).Decode(&fromUser)
		incomingHTML += fmt.Sprintf(`
			<div style="display:flex; justify-content:space-between; align-items:center; padding:10px; border-top:1px solid #3f4147;">
				<div style="display:flex; align-items:center; gap:12px;">
					<img src="%s" style="width:32px; height:32px; border-radius:50%%;">
					<div><b style="color:white;">%s</b><br><span style="font-size:12px; color:#949ba4;">Incoming Friend Request</span></div>
				</div>
				<div style="display:flex; gap:8px;">
					<button hx-post="/api/v1/users/@me/relationships/%s/accept" hx-target="#main-chat" style="background:#248046; color:white; border:none; padding:8px; border-radius:50%%; cursor:pointer;" title="Accept">✓</button>
					<button hx-post="/api/v1/users/@me/relationships/%s/reject" hx-target="#main-chat" style="background:#da373c; color:white; border:none; padding:8px; border-radius:50%%; cursor:pointer;" title="Decline">✕</button>
				</div>
			</div>
		`, fromUser.Avatar, fromUser.Username, req.ID.Hex(), req.ID.Hex())
	}

	// Fetch user details for friends list
	friendsHTML := ""
	if len(friends) == 0 {
		friendsHTML = `<div style="text-align:center; padding: 40px; color: #949ba4;">No one is around to play with Wumpus.</div>`
	}
	for _, friend := range friends {
		friendsHTML += fmt.Sprintf(`
			<div style="display:flex; justify-content:space-between; align-items:center; padding:10px; border-top:1px solid #3f4147;">
				<div style="display:flex; align-items:center; gap:12px;">
					<img src="%s" style="width:32px; height:32px; border-radius:50%%;">
					<div><b style="color:white;">%s</b></div>
				</div>
				<div style="display:flex; gap:8px;">
					<form hx-post="/api/v1/users/@me/channels" hx-target="#channels-container" onsubmit="event.preventDefault()">
						<input type="hidden" name="friend_id" value="%s">
						<button type="submit" style="background:#2b2d31; color:#dbdee1; border:none; padding:8px 12px; border-radius:4px; cursor:pointer;">Message</button>
					</form>
				</div>
			</div>
		`, friend.Avatar, friend.Username, friend.ID.Hex())
	}

	html := fmt.Sprintf(`
		<div style="display:flex; flex-direction:column; height:100%%; background:var(--bg-primary); color:var(--text-normal); width:100%%;">
			<!-- Header -->
			<div class="chat-header">
				<div style="font-weight:bold; display:flex; align-items:center; gap:8px; margin-right:16px;">
					<svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor"><path d="M12 14C14.2091 14 15.8333 12.3333 15.8333 10.1667C15.8333 8 14.2091 6.33333 12 6.33333C9.79086 6.33333 8.16667 8 8.16667 10.1667C8.16667 12.3333 9.79086 14 12 14ZM12 15C9.33333 15 4 16.3333 4 19V20H20V19C20 16.3333 14.6667 15 12 15Z"/></svg>
					Friends
				</div>
				<div style="width:1px; height:24px; background:var(--bg-tertiary);"></div>
                <div style="display:flex; gap: 8px;">
				    <button onclick="document.getElementById('tab-all').style.display='block'; document.getElementById('tab-pending').style.display='none'; document.getElementById('tab-add').style.display='none';" style="background:transparent; border:none; color:var(--text-normal); cursor:pointer; font-weight:500; font-size:16px; padding:4px 8px; border-radius:4px;">All</button>
				    <button onclick="document.getElementById('tab-all').style.display='none'; document.getElementById('tab-pending').style.display='block'; document.getElementById('tab-add').style.display='none';" style="background:transparent; border:none; color:var(--text-normal); cursor:pointer; font-weight:500; font-size:16px; padding:4px 8px; border-radius:4px;">Pending <span style="background:var(--danger); color:white; border-radius:50%%; padding:2px 6px; font-size:12px;">%d</span></button>
				    <button onclick="document.getElementById('tab-all').style.display='none'; document.getElementById('tab-pending').style.display='none'; document.getElementById('tab-add').style.display='block';" style="background:#248046; border:none; color:white; cursor:pointer; font-weight:500; font-size:14px; padding:4px 8px; border-radius:4px;">Add Friend</button>
                </div>
			</div>

			<!-- Tab Content -->
			<div style="padding:24px; flex:1; overflow-y:auto;">

				<!-- All Friends Tab -->
				<div id="tab-all" style="display:block;">
					<h3 style="font-size:12px; text-transform:uppercase; color:#949ba4; margin-bottom:16px;">All Friends - %d</h3>
					%s
				</div>

				<!-- Pending Requests Tab -->
				<div id="tab-pending" style="display:none;">
					<h3 style="font-size:12px; text-transform:uppercase; color:#949ba4; margin-bottom:16px;">Pending - %d</h3>
					%s
				</div>

				<!-- Add Friend Tab -->
				<div id="tab-add" style="display:none;">
					<h2 style="margin-top:0; margin-bottom: 8px; color:var(--header-primary); font-size: 16px; text-transform: uppercase;">Add Friend</h2>
					<p style="color:var(--header-secondary); font-size:14px; margin-bottom:16px;">You can add friends with their Discord username.</p>
					<form hx-post="/api/v1/users/@me/relationships" hx-target="#friend-msg" onsubmit="event.preventDefault()">
						<div style="display:flex; background:var(--bg-tertiary); border:1px solid rgba(0,0,0,0.3); border-radius:8px; padding:12px 16px; gap:16px; align-items:center;">
							<input type="text" name="friend_username" placeholder="You can add friends with their username." required style="flex:1; background:transparent; border:none; color:var(--text-normal); font-size:16px; outline:none;">
							<button type="submit" style="background:var(--brand); color:white; border:none; padding:8px 16px; border-radius:4px; font-weight:500; cursor:pointer;">Send Friend Request</button>
						</div>
					</form>
					<div id="friend-msg" style="margin-top: 10px; font-size:14px;"></div>
				</div>

			</div>
		</div>
	`, len(incomingReqs), len(friends), friendsHTML, len(incomingReqs), incomingHTML)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func SendFriendRequest(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(primitive.ObjectID)
	targetUsername := strings.TrimSpace(r.FormValue("friend_username"))

	var targetUser models.User
	err := db.UsersCollection.FindOne(context.Background(), bson.M{"username": targetUsername}).Decode(&targetUser)
	if err != nil {
		w.Write([]byte(`<span style="color:#da373c;">Hm, didn't work. Double check that the username is correct.</span>`))
		return
	}

	if targetUser.ID == userID {
		w.Write([]byte(`<span style="color:#da373c;">You can't add yourself as a friend.</span>`))
		return
	}

	// Check if already friends
	for _, friendID := range targetUser.Friends {
		if friendID == userID {
			w.Write([]byte(`<span style="color:#da373c;">You are already friends with that user!</span>`))
			return
		}
	}

	// Check if request already exists
	count, _ := db.FriendRequestsCollection.CountDocuments(context.Background(), bson.M{
		"from_user_id": userID,
		"to_user_id":   targetUser.ID,
		"status":       "pending",
	})
	if count > 0 {
		w.Write([]byte(`<span style="color:#da373c;">You've already sent a request to that user.</span>`))
		return
	}

	newRequest := models.FriendRequest{
		ID:         primitive.NewObjectID(),
		FromUserID: userID,
		ToUserID:   targetUser.ID,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	_, err = db.FriendRequestsCollection.InsertOne(context.Background(), newRequest)
	if err != nil {
		w.Write([]byte(`<span style="color:#da373c;">Failed to send request.</span>`))
		return
	}

	w.Write([]byte(`<span style="color:#248046;">Success! Your friend request to <b>` + targetUser.Username + `</b> was sent.</span>`))
}

func AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(primitive.ObjectID)
	requestIDHex := chi.URLParam(r, "requestID")
	requestID, err := primitive.ObjectIDFromHex(requestIDHex)
	if err != nil {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	var request models.FriendRequest
	err = db.FriendRequestsCollection.FindOne(context.Background(), bson.M{"_id": requestID, "to_user_id": userID}).Decode(&request)
	if err != nil {
		http.Error(w, "Request not found", http.StatusNotFound)
		return
	}

	// Add to each other's friends list using $addToSet to avoid duplicates
	_, err = db.UsersCollection.UpdateOne(context.Background(), bson.M{"_id": userID}, bson.M{"$addToSet": bson.M{"friends": request.FromUserID}})
	if err != nil {
		fmt.Println("Error adding friend for user: ", err)
	}
	_, err = db.UsersCollection.UpdateOne(context.Background(), bson.M{"_id": request.FromUserID}, bson.M{"$addToSet": bson.M{"friends": userID}})
	if err != nil {
		fmt.Println("Error adding friend for target user: ", err)
	}

	// Delete request
	_, err = db.FriendRequestsCollection.DeleteOne(context.Background(), bson.M{"_id": requestID})
	if err != nil {
		fmt.Println("Error deleting friend request: ", err)
	}

	// Re-render Friends view
	GetMe(w, r)
}

func RejectFriendRequest(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(primitive.ObjectID)
	requestIDHex := chi.URLParam(r, "requestID")
	requestID, err := primitive.ObjectIDFromHex(requestIDHex)
	if err != nil {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	// Delete request if the target is the user
	result, err := db.FriendRequestsCollection.DeleteOne(context.Background(), bson.M{"_id": requestID, "to_user_id": userID})
	if err != nil {
		fmt.Println("Error rejecting friend request: ", err)
	}
	if result.DeletedCount == 0 {
		fmt.Println("No friend request deleted. ID mismatch or unauthorized.")
	}

	// Re-render Friends view
	GetMe(w, r)
}

func CreateDirectMessage(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(primitive.ObjectID)
	friendIDHex := r.FormValue("friend_id")
	friendID, _ := primitive.ObjectIDFromHex(friendIDHex)

	// Check if a DM channel already exists between these two
	var existingChannel models.DirectChannel
	err := db.ChannelsCollection.FindOne(context.Background(), bson.M{
		"type":         "direct",
		"participants": bson.M{"$all": []primitive.ObjectID{userID, friendID}},
	}).Decode(&existingChannel)

	var channelID primitive.ObjectID
	if err == mongo.ErrNoDocuments {
		// Create new DM channel
		newChannel := models.DirectChannel{
			ID:           primitive.NewObjectID(),
			Type:         "direct",
			Participants: []primitive.ObjectID{userID, friendID},
		}
		db.ChannelsCollection.InsertOne(context.Background(), newChannel)
		channelID = newChannel.ID
	} else {
		channelID = existingChannel.ID
	}

	// Simulate Discord logic: switch inner sidebar and open chat
	// Let's do a redirect to trigger the channel load in the main-chat via HX-Redirect or a script
	w.Header().Set("HX-Trigger", fmt.Sprintf(`{"openChannel": "%s"}`, channelID.Hex()))

	// Temporarily return a message. A more complete clone would render the DM list in the inner sidebar.
	w.Write([]byte(`<script>htmx.ajax('GET', '/api/v1/channels/` + channelID.Hex() + `', {target:'#main-chat'});</script>`))
}
