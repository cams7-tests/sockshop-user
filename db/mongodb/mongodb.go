package mongodb

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/microservices-demo/user/users"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	mongoUser string
	mongoPass string
	mongoHost string
	mongoDatabase = "users"
	//ErrInvalidHexID represents a entity id that is not a valid bson ObjectID
	ErrInvalidHexID = errors.New("Invalid Id Hex")
)

func init() {
	flag.StringVar(&mongoUser, "mongo-user", os.Getenv("MONGO_USER"), "Mongo user")
	flag.StringVar(&mongoPass, "mongo-password", os.Getenv("MONGO_PASS"), "Mongo password")
	flag.StringVar(&mongoHost, "mongo-host", os.Getenv("MONGO_HOST"), "Mongo host")
}

// Mongo meets the Database interface requirements
type Mongo struct {
	//Session is a MongoDB Session
	Session *mgo.Session
}

// Init MongoDB
func (mongo *Mongo) Init() error {
	mongoUrl := getURL()
	var err error
	mongo.Session, err = mgo.DialWithTimeout(mongoUrl.String(), time.Duration(5)*time.Second)
	if err != nil {
		return err
	}
	return mongo.EnsureIndexes()
}

// MongoUser is a wrapper for the users
type MongoUser struct {
	users.User `bson:",inline"`
	ID         bson.ObjectId   `bson:"_id"`
	AddressIDs []bson.ObjectId `bson:"addresses"`
	CardIDs    []bson.ObjectId `bson:"cards"`
}

// New Returns a new MongoUser
func New() MongoUser {
	user := users.New()
	return MongoUser{
		User:       user,
		AddressIDs: make([]bson.ObjectId, 0),
		CardIDs:    make([]bson.ObjectId, 0),
	}
}

// AddUserIDs adds userID as string to user
func (mongoUser *MongoUser) AddUserIDs() {
	if mongoUser.User.Addresses == nil {
		mongoUser.User.Addresses = make([]users.Address, 0)
	}
	for _, id := range mongoUser.AddressIDs {
		mongoUser.User.Addresses = append(mongoUser.User.Addresses, users.Address{
			ID: id.Hex(),
		})
	}
	if mongoUser.User.Cards == nil {
		mongoUser.User.Cards = make([]users.Card, 0)
	}
	for _, id := range mongoUser.CardIDs {
		mongoUser.User.Cards = append(mongoUser.User.Cards, users.Card{ID: id.Hex()})
	}
	mongoUser.User.UserID = mongoUser.ID.Hex()
}

// MongoAddress is a wrapper for Address
type MongoAddress struct {
	users.Address `bson:",inline"`
	ID            bson.ObjectId `bson:"_id"`
}

// AddID ObjectID as string
func (mongoAddress *MongoAddress) AddID() {
	mongoAddress.Address.ID = mongoAddress.ID.Hex()
}

// MongoCard is a wrapper for Card
type MongoCard struct {
	users.Card `bson:",inline"`
	ID         bson.ObjectId `bson:"_id"`
}

// AddID ObjectID as string
func (mongoCard *MongoCard) AddID() {
	mongoCard.Card.ID = mongoCard.ID.Hex()
}

// CreateUser Insert user to MongoDB, including connected addresses and cards, update passed in user with Ids
func (mongo *Mongo) CreateUser(user *users.User) error {
	session := mongo.Session.Copy()
	defer session.Close()
	id := bson.NewObjectId()
	mongoUser := New()
	mongoUser.User = *user
	mongoUser.ID = id
	var carderr error
	var addrerr error
	mongoUser.CardIDs, carderr = mongo.createCards(user.Cards)
	mongoUser.AddressIDs, addrerr = mongo.createAddresses(user.Addresses)
	c := session.DB("").C("customers")
	_, err := c.UpsertId(mongoUser.ID, mongoUser)
	if err != nil {
		// Gonna clean up if we can, ignore error
		// because the user save error takes precedence.
		mongo.cleanAttributes(mongoUser)
		return err
	}
	mongoUser.User.UserID = mongoUser.ID.Hex()
	// Cheap err for attributes
	if carderr != nil || addrerr != nil {
		return fmt.Errorf("%v %v", carderr, addrerr)
	}
	*user = mongoUser.User
	return nil
}

func (mongo *Mongo) createCards(cards []users.Card) ([]bson.ObjectId, error) {
	session := mongo.Session.Copy()
	defer session.Close()
	ids := make([]bson.ObjectId, 0)
	defer session.Close()
	for k, card := range cards {
		id := bson.NewObjectId()
		mongoCard := MongoCard{Card: card, ID: id}
		c := session.DB("").C("cards")
		_, err := c.UpsertId(mongoCard.ID, mongoCard)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
		cards[k].ID = id.Hex()
	}
	return ids, nil
}

func (mongo *Mongo) createAddresses(addresses []users.Address) ([]bson.ObjectId, error) {
	ids := make([]bson.ObjectId, 0)
	session := mongo.Session.Copy()
	defer session.Close()
	for k, address := range addresses {
		id := bson.NewObjectId()
		mongoAddress := MongoAddress{Address: address, ID: id}
		c := session.DB("").C("addresses")
		_, err := c.UpsertId(mongoAddress.ID, mongoAddress)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
		addresses[k].ID = id.Hex()
	}
	return ids, nil
}

func (mongo *Mongo) cleanAttributes(mongoUser MongoUser) error {
	session := mongo.Session.Copy()
	defer session.Close()
	c := session.DB("").C("addresses")
	_, err := c.RemoveAll(bson.M{"_id": bson.M{"$in": mongoUser.AddressIDs}})
	c = session.DB("").C("cards")
	_, err = c.RemoveAll(bson.M{"_id": bson.M{"$in": mongoUser.CardIDs}})
	return err
}

func (mongo *Mongo) appendAttributeId(attr string, id bson.ObjectId, userid string) error {
	session := mongo.Session.Copy()
	defer session.Close()
	c := session.DB("").C("customers")
	return c.Update(bson.M{"_id": bson.ObjectIdHex(userid)},
		bson.M{"$addToSet": bson.M{attr: id}})
}

func (mongo *Mongo) removeAttributeId(attr string, id bson.ObjectId, userid string) error {
	session := mongo.Session.Copy()
	defer session.Close()
	c := session.DB("").C("customers")
	return c.Update(bson.M{"_id": bson.ObjectIdHex(userid)},
		bson.M{"$pull": bson.M{attr: id}})
}

// GetUserByName Get user by their name
func (mongo *Mongo) GetUserByName(name string) (users.User, error) {
	session := mongo.Session.Copy()
	defer session.Close()
	c := session.DB("").C("customers")
	mongoUser := New()
	err := c.Find(bson.M{"username": name}).One(&mongoUser)
	mongoUser.AddUserIDs()
	return mongoUser.User, err
}

// GetUser Get user by their object id
func (mongo *Mongo) GetUser(id string) (users.User, error) {
	session := mongo.Session.Copy()
	defer session.Close()
	if !bson.IsObjectIdHex(id) {
		return users.New(), errors.New("Invalid Id Hex")
	}
	c := session.DB("").C("customers")
	mongoUser := New()
	err := c.FindId(bson.ObjectIdHex(id)).One(&mongoUser)
	mongoUser.AddUserIDs()
	return mongoUser.User, err
}

// GetUsers Get all users
func (mongo *Mongo) GetUsers() ([]users.User, error) {
	// TODO: add paginations
	session := mongo.Session.Copy()
	defer session.Close()
	c := session.DB("").C("customers")
	var mongoUsers []MongoUser
	err := c.Find(nil).All(&mongoUsers)
	us := make([]users.User, 0)
	for _, mongoUser := range mongoUsers {
		mongoUser.AddUserIDs()
		us = append(us, mongoUser.User)
	}
	return us, err
}

// GetUserAttributes given a user, load all cards and addresses connected to that user
func (mongo *Mongo) GetUserAttributes(user *users.User) error {
	session := mongo.Session.Copy()
	defer session.Close()
	ids := make([]bson.ObjectId, 0)
	for _, address := range user.Addresses {
		if !bson.IsObjectIdHex(address.ID) {
			return ErrInvalidHexID
		}
		ids = append(ids, bson.ObjectIdHex(address.ID))
	}
	var mongoAddresses []MongoAddress
	c := session.DB("").C("addresses")
	err := c.Find(bson.M{"_id": bson.M{"$in": ids}}).All(&mongoAddresses)
	if err != nil {
		return err
	}
	addresses := make([]users.Address, 0)
	for _, mongoAddress := range mongoAddresses {
		mongoAddress.Address.ID = mongoAddress.ID.Hex()
		addresses = append(addresses, mongoAddress.Address)
	}
	user.Addresses = addresses

	ids = make([]bson.ObjectId, 0)
	for _, card := range user.Cards {
		if !bson.IsObjectIdHex(card.ID) {
			return ErrInvalidHexID
		}
		ids = append(ids, bson.ObjectIdHex(card.ID))
	}
	var mongoCards []MongoCard
	c = session.DB("").C("cards")
	err = c.Find(bson.M{"_id": bson.M{"$in": ids}}).All(&mongoCards)
	if err != nil {
		return err
	}

	cards := make([]users.Card, 0)
	for _, mongoCard := range mongoCards {
		mongoCard.Card.ID = mongoCard.ID.Hex()
		cards = append(cards, mongoCard.Card)
	}
	user.Cards = cards
	return nil
}

// GetCard Gets card by objects Id
func (mongo *Mongo) GetCard(id string) (users.Card, error) {
	session := mongo.Session.Copy()
	defer session.Close()
	if !bson.IsObjectIdHex(id) {
		return users.Card{}, errors.New("Invalid Id Hex")
	}
	c := session.DB("").C("cards")
	mongoCard := MongoCard{}
	err := c.FindId(bson.ObjectIdHex(id)).One(&mongoCard)
	mongoCard.AddID()
	return mongoCard.Card, err
}

// GetCards Gets all cards
func (mongo *Mongo) GetCards() ([]users.Card, error) {
	// TODO: add pagination
	session := mongo.Session.Copy()
	defer session.Close()
	c := session.DB("").C("cards")
	var mongoCards []MongoCard
	err := c.Find(nil).All(&mongoCards)
	cards := make([]users.Card, 0)
	for _, mongoCard := range mongoCards {
		mongoCard.AddID()
		cards = append(cards, mongoCard.Card)
	}
	return cards, err
}

// CreateCard adds card to MongoDB
func (mongo *Mongo) CreateCard(card *users.Card, userid string) error {
	if userid != "" && !bson.IsObjectIdHex(userid) {
		return errors.New("Invalid Id Hex")
	}
	session := mongo.Session.Copy()
	defer session.Close()
	c := session.DB("").C("cards")
	id := bson.NewObjectId()
	mongoCard := MongoCard{Card: *card, ID: id}
	_, err := c.UpsertId(mongoCard.ID, mongoCard)
	if err != nil {
		return err
	}
	// Address for anonymous user
	if userid != "" {
		err = mongo.appendAttributeId("cards", mongoCard.ID, userid)
		if err != nil {
			return err
		}
	}
	mongoCard.AddID()
	*card = mongoCard.Card
	return err
}

// GetAddress Gets an address by object Id
func (mongo *Mongo) GetAddress(id string) (users.Address, error) {
	session := mongo.Session.Copy()
	defer session.Close()
	if !bson.IsObjectIdHex(id) {
		return users.Address{}, errors.New("Invalid Id Hex")
	}
	c := session.DB("").C("addresses")
	mongoAddress := MongoAddress{}
	err := c.FindId(bson.ObjectIdHex(id)).One(&mongoAddress)
	mongoAddress.AddID()
	return mongoAddress.Address, err
}

// GetAddresses gets all addresses
func (mongo *Mongo) GetAddresses() ([]users.Address, error) {
	// TODO: add pagination
	session := mongo.Session.Copy()
	defer session.Close()
	c := session.DB("").C("addresses")
	var mongoAddresses []MongoAddress
	err := c.Find(nil).All(&mongoAddresses)
	addresses := make([]users.Address, 0)
	for _, mongoAddress := range mongoAddresses {
		mongoAddress.AddID()
		addresses = append(addresses, mongoAddress.Address)
	}
	return addresses, err
}

// CreateAddress Inserts Address into MongoDB
func (mongo *Mongo) CreateAddress(address *users.Address, userid string) error {
	if userid != "" && !bson.IsObjectIdHex(userid) {
		return errors.New("Invalid Id Hex")
	}
	session := mongo.Session.Copy()
	defer session.Close()
	c := session.DB("").C("addresses")
	id := bson.NewObjectId()
	mongoAddress := MongoAddress{Address: *address, ID: id}
	_, err := c.UpsertId(mongoAddress.ID, mongoAddress)
	if err != nil {
		return err
	}
	// Address for anonymous user
	if userid != "" {
		err = mongo.appendAttributeId("addresses", mongoAddress.ID, userid)
		if err != nil {
			return err
		}
	}
	mongoAddress.AddID()
	*address = mongoAddress.Address
	return err
}

// CreateAddress Inserts Address into MongoDB
func (mongo *Mongo) Delete(entity, id string) error {
	if !bson.IsObjectIdHex(id) {
		return errors.New("Invalid Id Hex")
	}
	session := mongo.Session.Copy()
	defer session.Close()
	c := session.DB("").C(entity)
	if entity == "customers" {
		user, err := mongo.GetUser(id)
		if err != nil {
			return err
		}
		aids := make([]bson.ObjectId, 0)
		for _, address := range user.Addresses {
			aids = append(aids, bson.ObjectIdHex(address.ID))
		}
		cids := make([]bson.ObjectId, 0)
		for _, c := range user.Cards {
			cids = append(cids, bson.ObjectIdHex(c.ID))
		}
		ac := session.DB("").C("addresses")
		ac.RemoveAll(bson.M{"_id": bson.M{"$in": aids}})
		cc := session.DB("").C("cards")
		cc.RemoveAll(bson.M{"_id": bson.M{"$in": cids}})
	} else {
		c := session.DB("").C("customers")
		c.UpdateAll(bson.M{},
			bson.M{"$pull": bson.M{entity: bson.ObjectIdHex(id)}})
	}
	return c.Remove(bson.M{"_id": bson.ObjectIdHex(id)})
}

func getURL() url.URL {
	//mongodb://[mongoUser]:[mongoPass]@[mongoHost]:27017/[mongoDatabase]
	mongoUrl := url.URL{
		Scheme: "mongodb",
		Host:   mongoHost,
		Path:   mongoDatabase,
	}
	if mongoUser != "" {
		userAndPassword := url.UserPassword(mongoUser, mongoPass)
		mongoUrl.User = userAndPassword
	}
	return mongoUrl
}

// EnsureIndexes ensures username is unique
func (mongo *Mongo) EnsureIndexes() error {
	session := mongo.Session.Copy()
	defer session.Close()
	i := mgo.Index{
		Key:        []string{"username"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     false,
	}
	c := session.DB("").C("customers")
	return c.EnsureIndex(i)
}

func (mongo *Mongo) Ping() error {
	session := mongo.Session.Copy()
	defer session.Close()
	return session.Ping()
}
