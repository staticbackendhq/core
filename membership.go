package staticbackend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"staticbackend/cache"
	"staticbackend/email"
	"staticbackend/internal"
	"staticbackend/middleware"
	"staticbackend/oauth"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"

	"github.com/gbrlsnchs/jwt/v3"
)

type membership struct {
	volatile *cache.Cache
}

func (m *membership) emailExists(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(r.URL.Query().Get("e"))
	if len(email) == 0 {
		respond(w, http.StatusOK, false)
		return
	}

	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		return
	}

	curDB := client.Database(conf.Name)
	exists, err := m.checkEmailExists(curDB, email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, exists)
}

func (m *membership) checkEmailExists(db *mongo.Database, email string) (bool, error) {
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	count, err := db.Collection("sb_tokens").CountDocuments(ctx, bson.M{"email": email})
	if err != nil {
		return false, err
	}

	return count == 1, nil
}

func (m *membership) login(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		return
	}

	db := client.Database(conf.Name)

	var l internal.Login
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	l.Email = strings.ToLower(l.Email)

	tok, err := m.validateUserPassword(db, l.Email, l.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jwtToken, err := m.loginSetup(tok, conf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, jwtToken)
}

func (m *membership) loginSetup(tok *internal.Token, conf internal.BaseConfig) (string, error) {
	token := fmt.Sprintf("%s|%s", tok.ID.Hex(), tok.Token)

	// get their JWT
	jwtBytes, err := m.getJWT(token)
	if err != nil {
		return "", err
	}

	auth := internal.Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      tok.Role,
		Token:     tok.Token,
	}

	if err := m.volatile.SetTyped(token, auth); err != nil {
		return "", err
	}
	if err := m.volatile.SetTyped("base:"+token, conf); err != nil {
		return "", err
	}

	return string(jwtBytes), nil
}

func (m *membership) validateUserPassword(db *mongo.Database, email, password string) (*internal.Token, error) {
	email = strings.ToLower(email)

	ctx := context.Background()
	sr := db.Collection("sb_tokens").FindOne(ctx, bson.M{"email": email})

	var tok internal.Token
	if err := sr.Decode(&tok); err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(tok.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid email/password")
	}

	return &tok, nil
}

func (m *membership) register(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		log.Println("invalid StaticBackend key")
		return
	}

	db := client.Database(conf.Name)

	var l internal.Login
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	l.Email = strings.ToLower(l.Email)

	// make sure this email does not exists
	count, err := db.Collection("sb_tokens").CountDocuments(context.Background(), bson.M{"email": l.Email})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if count > 0 {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}

	_, tok, err := m.createAccountAndUser(db, l.Email, l.Password, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jwtToken, err := m.loginSetup(&tok, conf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, jwtToken)
}

func (m *membership) createAccountAndUser(db *mongo.Database, email, password string, role int) ([]byte, internal.Token, error) {
	acctID := primitive.NewObjectID()

	a := internal.Account{
		ID:    acctID,
		Email: email,
	}

	ctx := context.Background()
	_, err := db.Collection("sb_accounts").InsertOne(ctx, a)
	if err != nil {
		return nil, internal.Token{}, err
	}

	jwtBytes, tok, err := m.createUser(db, acctID, email, password, role)
	if err != nil {
		return nil, internal.Token{}, err
	}
	return jwtBytes, tok, nil
}

func (m *membership) createUser(db *mongo.Database, accountID primitive.ObjectID, email, password string, role int) ([]byte, internal.Token, error) {
	ctx := context.Background()

	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, internal.Token{}, err
	}

	tok := internal.Token{
		ID:        primitive.NewObjectID(),
		AccountID: accountID,
		Email:     email,
		Token:     primitive.NewObjectID().Hex(),
		Password:  string(b),
		Role:      role,
	}

	_, err = db.Collection("sb_tokens").InsertOne(ctx, tok)
	if err != nil {
		return nil, tok, err
	}

	token := fmt.Sprintf("%s|%s", tok.ID.Hex(), tok.Token)

	// Get their JWT
	jwtBytes, err := m.getJWT(token)
	if err != nil {
		return nil, tok, err
	}

	auth := internal.Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      role,
		Token:     tok.Token,
	}
	if err := m.volatile.SetTyped(token, auth); err != nil {
		return nil, tok, err
	}

	return jwtBytes, tok, nil
}

func (m *membership) setRole(w http.ResponseWriter, r *http.Request) {
	conf, a, err := middleware.Extract(r, true)
	if err != nil || a.Role < 100 {
		http.Error(w, "insufficient priviledges", http.StatusUnauthorized)
		return
	}

	var data = new(struct {
		Email string `json:"email"`
		Role  int    `json:"role"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := client.Database(conf.Name)

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	filter := bson.M{"email": data.Email}
	update := bson.M{"$set": bson.M{"role": data.Role}}
	if _, err := db.Collection("sb_tokens").UpdateOne(ctx, filter, update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func (m *membership) setPassword(w http.ResponseWriter, r *http.Request) {
	conf, a, err := middleware.Extract(r, true)
	if err != nil || a.Role < 100 {
		http.Error(w, "insufficient priviledges", http.StatusUnauthorized)
		return
	}

	var data = new(struct {
		Email       string `json:"email"`
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := client.Database(conf.Name)

	tok, err := m.validateUserPassword(db, data.Email, data.OldPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	newpw, err := bcrypt.GenerateFromPassword([]byte(data.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	filter := bson.M{"_id": tok.ID}
	update := bson.M{"$set": bson.M{"pw": string(newpw)}}
	if _, err := db.Collection("sb_tokens").UpdateOne(ctx, filter, update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func (m *membership) resetPassword(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		return
	}

	db := client.Database(conf.Name)

	var data = new(struct {
		Email string `json:"email"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filter := bson.M{"email": strings.ToLower(data.Email)}
	count, err := db.Collection("sb_tokens").CountDocuments(context.Background(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if count == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	code := randStringRunes(6)
	update := bson.M{"%set": bson.M{"sb_reset_code": code}}
	if _, err := db.Collection("sb_tokens").UpdateOne(context.Background(), filter, update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//TODO: have HTML template for those
	body := fmt.Sprintf(`Your reset code is: %s`, code)

	ed := internal.SendMailData{
		From:     FromEmail,
		FromName: FromName,
		To:       data.Email,
		Subject:  "Your password reset code",
		HTMLBody: body,
		TextBody: email.StripHTML(body),
	}
	if err := emailer.Send(ed); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func (m *membership) changePassword(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		return
	}

	db := client.Database(conf.Name)

	var data = new(struct {
		Email    string `json:"email"`
		Code     string `json:"code"`
		Password string `json:"password"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filter := bson.M{"email": strings.ToLower(data.Email), "sb_reset_code": data.Code}
	var tok internal.Token
	sr := db.Collection("sb_tokens").FindOne(context.Background(), filter)
	if err := sr.Decode(&tok); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newpw, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	filter = bson.M{internal.FieldID: tok.ID}
	update := bson.M{"$set": bson.M{"pw": string(newpw)}}
	if _, err := db.Collection("sb_tokens").UpdateOne(ctx, filter, update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func (m *membership) getJWT(token string) ([]byte, error) {
	now := time.Now()
	pl := internal.JWTPayload{
		Payload: jwt.Payload{
			Issuer:         "StaticBackend",
			ExpirationTime: jwt.NumericDate(now.Add(12 * time.Hour)),
			NotBefore:      jwt.NumericDate(now.Add(30 * time.Minute)),
			IssuedAt:       jwt.NumericDate(now),
			JWTID:          primitive.NewObjectID().Hex(),
		},
		Token: token,
	}

	return jwt.Sign(pl, internal.HashSecret)

}

func (m *membership) sudoGetTokenFromAccountID(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := client.Database(conf.Name)

	id := ""

	_, r.URL.Path = ShiftPath(r.URL.Path)
	id, r.URL.Path = ShiftPath(r.URL.Path)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filter := bson.M{internal.FieldAccountID: oid}
	ctx := context.Background()

	opt := options.Find()
	opt.SetLimit(1)
	opt.SetSort(bson.M{internal.FieldID: 1})

	cur, err := db.Collection("sb_tokens").Find(ctx, filter, opt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)

	var tok internal.Token
	if cur.Next(ctx) {
		if err := cur.Decode(&tok); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if len(tok.Token) == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	token := fmt.Sprintf("%s|%s", tok.ID.Hex(), tok.Token)

	jwtBytes, err := m.getJWT(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	auth := internal.Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      tok.Role,
		Token:     tok.Token,
	}
	if err := m.volatile.SetTyped(token, auth); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, string(jwtBytes))
}

type OauthFlowState struct {
	PublicKey   string `json:"pk"`
	DBName      string `json:"dbName"`
	Provider    string `json:"provider"`
	RedirectURL string `json:"redirectUrl"`
}

func (m *membership) externalLogin(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// /oauth/login/:provider
	provider := getURLPart(r.URL.Path, 3)

	curDB := client.Database(conf.Name)

	data, err := oauth.GetConfig(curDB, provider)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if len(data.ClientID) == 0 {
		http.Error(w, "missing configuration for this external login provider", http.StatusBadRequest)
		return
	}

	//TODO: Change the LOCAL_STORAGE_URL here to a new Env Var, like APP_URL
	oauthConf := &oauth2.Config{
		ClientID:     data.ClientID,
		ClientSecret: data.ClientSecret,
		Scopes:       data.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  data.AuthURL,
			TokenURL: data.TokenURL,
		},
		RedirectURL: fmt.Sprintf("%s/oauth/code", os.Getenv("LOCAL_STORAGE_URL")),
	}

	state := OauthFlowState{
		PublicKey: conf.ID.Hex(),
		DBName:    conf.Name,
		Provider:  provider,
	}

	// we'll need this when the oauth provider call our callback
	if err := m.volatile.SetTyped(fmt.Sprint("oauth:%s", conf.ID.Hex()), state); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	url := oauthConf.AuthCodeURL(conf.ID.Hex(), oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (m *membership) exchangeCode(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	var flowState OauthFlowState
	if err := m.volatile.GetTyped(fmt.Sprintf("oauth:%s", state), &flowState); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	curDB := client.Database(flowState.DBName)

	data, err := oauth.GetConfig(curDB, flowState.Provider)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	oauthConf := &oauth2.Config{
		ClientID:     data.ClientID,
		ClientSecret: data.ClientSecret,
		Scopes:       data.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  data.AuthURL,
			TokenURL: data.TokenURL,
		},
		RedirectURL: fmt.Sprintf("%s/oauth/code", os.Getenv("LOCAL_STORAGE_URL")),
	}

	ctx := context.Background()
	tok, err := oauthConf.Exchange(ctx, code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := oauthConf.Client(ctx, tok)

	grabber := oauth.GetInfoGrabber(data.Provider)
	if grabber == nil {
		http.Error(w, "cannot find implementation for this provider", http.StatusBadRequest)
		return
	}

	info, err := grabber.Get(client, tok.AccessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	info.Email = strings.ToLower(info.Email)

	exists, err := m.checkEmailExists(curDB, info.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var userToken internal.Token

	if exists {
		t, err := internal.FindTokenByEmail(curDB, info.Email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		userToken = t
	} else {
		_, newToken, err := m.createAccountAndUser(curDB, info.Email, "el", 1)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		userToken = newToken
	}

	// the BaseConfig has to be in the cache...
	// TODO: there's a huge clean-up that has to be done with all the cache / volatile concepts
	var conf internal.BaseConfig
	if err := volatile.GetTyped(flowState.PublicKey, &conf); err == nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jwtToken, err := m.loginSetup(&userToken, conf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	u, err := url.Parse(flowState.RedirectURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u.Query().Add("token", jwtToken)

	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
}
