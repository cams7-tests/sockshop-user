package mongodb

import (
	"flag"

	"context"
	"os"
	"time"

	"github.com/microservices-demo/user/users"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	mongoConnection string
	mongoDatabase         = "users"
	totalRows       int64 = 100
)

func init() {
	flag.StringVar(&mongoConnection, "mongo-connection-string", os.Getenv("MONGODB_CONNECTION_STRING"), "Mongo connection string")
}

// Mongo meets the Database interface requirements
type Mongo struct {
	Client *mongo.Client
}

// Init MongoDB
func (m *Mongo) Init() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoConnection))
	m.Client = client
	if err != nil {
		return err
	}

	return nil
}

// MongoUser is a wrapper for the users
type MongoUser struct {
	users.User `bson:",inline"`
	ID         primitive.ObjectID   `bson:"_id"`
	AddressIDs []primitive.ObjectID `bson:"addresses"`
	CardIDs    []primitive.ObjectID `bson:"cards"`
}

// New Returns a new MongoUser
func New() MongoUser {
	user := users.New()
	return MongoUser{
		User:       user,
		AddressIDs: make([]primitive.ObjectID, 0),
		CardIDs:    make([]primitive.ObjectID, 0),
	}
}

// AddUserIDs adds userID as string to user
func (mu *MongoUser) AddUserIDs() {
	if mu.User.Addresses == nil {
		mu.User.Addresses = make([]users.Address, 0)
	}
	for _, id := range mu.AddressIDs {
		mu.User.Addresses = append(mu.User.Addresses, users.Address{
			ID: id.Hex(),
		})
	}
	if mu.User.Cards == nil {
		mu.User.Cards = make([]users.Card, 0)
	}
	for _, id := range mu.CardIDs {
		mu.User.Cards = append(mu.User.Cards, users.Card{ID: id.Hex()})
	}
	mu.User.UserID = mu.ID.Hex()
}

// MongoAddress is a wrapper for Address
type MongoAddress struct {
	users.Address `bson:",inline"`
	ID            primitive.ObjectID `bson:"_id"`
}

// AddID ObjectID as string
func (ma *MongoAddress) AddID() {
	ma.Address.ID = ma.ID.Hex()
}

// MongoCard is a wrapper for Card
type MongoCard struct {
	users.Card `bson:",inline"`
	ID         primitive.ObjectID `bson:"_id"`
}

// AddID ObjectID as string
func (mc *MongoCard) AddID() {
	mc.Card.ID = mc.ID.Hex()
}

// CreateUser Insert user to MongoDB, including connected addresses and cards, update passed in user with Ids
func (m *Mongo) CreateUser(user *users.User) error {
	id := primitive.NewObjectID()
	mu := New()
	mu.User = *user
	mu.ID = id

	var err error
	mu.CardIDs, err = m.createCards(user.Cards)
	if err != nil {
		return err
	}

	mu.AddressIDs, err = m.createAddresses(user.Addresses)
	if err != nil {
		return err
	}

	collection := m.Client.Database(mongoDatabase).Collection("customers")
	_, err = collection.InsertOne(context.Background(), mu)
	if err != nil {
		return err
	}
	mu.User.UserID = mu.ID.Hex()
	*user = mu.User
	return nil
}

func (m *Mongo) createCards(cards []users.Card) ([]primitive.ObjectID, error) {
	ids := make([]primitive.ObjectID, 0)
	for i, card := range cards {
		id := primitive.NewObjectID()
		mc := MongoCard{Card: card, ID: id}
		collection := m.Client.Database(mongoDatabase).Collection("cards")
		_, err := collection.InsertOne(context.Background(), mc)
		if err != nil {
			m.cleanCardsAttr(ids)
			return ids, err
		}
		ids = append(ids, id)
		cards[i].ID = id.Hex()
	}
	return ids, nil
}

func (m *Mongo) createAddresses(addresses []users.Address) ([]primitive.ObjectID, error) {
	ids := make([]primitive.ObjectID, 0)
	for i, address := range addresses {
		id := primitive.NewObjectID()
		ma := MongoAddress{Address: address, ID: id}
		collection := m.Client.Database(mongoDatabase).Collection("addresses")
		_, err := collection.InsertOne(context.Background(), ma)
		if err != nil {
			m.cleanAddressesAttr(ids)
			return ids, err
		}
		ids = append(ids, id)
		addresses[i].ID = id.Hex()
	}
	return ids, nil
}

func (m *Mongo) cleanAddressesAttr(addressesIds []primitive.ObjectID) error {
	collection := m.Client.Database(mongoDatabase).Collection("addresses")
	_, err := collection.DeleteMany(context.Background(), bson.M{"_id": bson.M{"$in": addressesIds}})
	return err
}

func (m *Mongo) cleanCardsAttr(cardIds []primitive.ObjectID) error {
	collection := m.Client.Database(mongoDatabase).Collection("cards")
	_, err := collection.DeleteMany(context.Background(), bson.M{"_id": bson.M{"$in": cardIds}})
	return err
}

func (m *Mongo) appendAttributeId(attr string, attrId primitive.ObjectID, userId string) error {
	id, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return err
	}

	collection := m.Client.Database(mongoDatabase).Collection("customers")
	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": bson.M{"$eq": id}}, bson.M{"$addToSet": bson.M{attr: attrId}})
	return err
}

func (m *Mongo) removeAttributeId(attr string, attrId primitive.ObjectID, userId string) error {
	id, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return err
	}

	collection := m.Client.Database(mongoDatabase).Collection("customers")
	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": bson.M{"$eq": id}}, bson.M{"$pull": bson.M{attr: attrId}})
	return err
}

// GetUserByName Get user by their name
func (m *Mongo) GetUserByName(username string) (users.User, error) {
	collection := m.Client.Database(mongoDatabase).Collection("customers")
	var mu MongoUser
	err := collection.FindOne(context.Background(), bson.M{"username": bson.M{"$eq": username}}).Decode(&mu)
	if err == nil {
		mu.AddUserIDs()
	}
	return mu.User, err
}

// GetUser Get user by their object id
func (m *Mongo) GetUser(id string) (users.User, error) {
	userId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return users.User{}, err
	}

	collection := m.Client.Database(mongoDatabase).Collection("customers")
	var mu MongoUser
	err = collection.FindOne(context.Background(), bson.M{"_id": bson.M{"$eq": userId}}).Decode(&mu)
	if err == nil {
		mu.AddUserIDs()
	}
	return mu.User, err
}

// GetUsers Get all users
func (m *Mongo) GetUsers() ([]users.User, error) {
	// TODO: add paginations
	collection := m.Client.Database(mongoDatabase).Collection("customers")
	findOptions := options.Find()
	findOptions.SetLimit(totalRows)
	cur, err := collection.Find(context.Background(), bson.D{{}}, findOptions)
	if err != nil {
		return []users.User{}, err
	}
	defer cur.Close(context.Background())

	us := make([]users.User, 0)
	for cur.Next(context.Background()) {
		var mu MongoUser
		err := cur.Decode(&mu)
		if err != nil {
			return []users.User{}, err
		}
		mu.AddUserIDs()
		us = append(us, mu.User)
	}

	return us, nil
}

// GetUserAttributes given a user, load all cards and addresses connected to that user
func (m *Mongo) GetUserAttributes(user *users.User) error {
	findOptions := options.Find()
	findOptions.SetLimit(totalRows)

	ids := make([]primitive.ObjectID, 0)
	for _, address := range user.Addresses {
		addressId, err := primitive.ObjectIDFromHex(address.ID)
		if err != nil {
			return err
		}
		ids = append(ids, addressId)
	}

	collection := m.Client.Database(mongoDatabase).Collection("addresses")
	cur, err := collection.Find(context.Background(), bson.M{"_id": bson.M{"$in": ids}}, findOptions)
	if err != nil {
		return err
	}
	defer cur.Close(context.Background())

	addresses := make([]users.Address, 0)
	for cur.Next(context.Background()) {
		var ma MongoAddress
		err := cur.Decode(&ma)
		if err != nil {
			return err
		}
		ma.Address.ID = ma.ID.Hex()
		addresses = append(addresses, ma.Address)
	}	
	user.Addresses = addresses

	ids = make([]primitive.ObjectID, 0)
	for _, card := range user.Cards {
		cardId, err := primitive.ObjectIDFromHex(card.ID)
		if err != nil {
			return err
		}
		ids = append(ids, cardId)
	}

	collection = m.Client.Database(mongoDatabase).Collection("cards")
	cur, err = collection.Find(context.Background(), bson.M{"_id": bson.M{"$in": ids}}, findOptions)
	if err != nil {
		return err
	}
	defer cur.Close(context.Background())

	cards := make([]users.Card, 0)
	for cur.Next(context.Background()) {
		var mc MongoCard
		err := cur.Decode(&mc)
		if err != nil {
			return err
		}
		mc.Card.ID = mc.ID.Hex()
		cards = append(cards, mc.Card)
	}
	user.Cards = cards

	return nil
}

// GetCard Gets card by objects Id
func (m *Mongo) GetCard(id string) (users.Card, error) {
	cardId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return users.Card{}, err
	}

	collection := m.Client.Database(mongoDatabase).Collection("cards")
	var mc MongoCard
	err = collection.FindOne(context.Background(), bson.M{"_id": bson.M{"$eq": cardId}}).Decode(&mc)
	if err == nil {
		mc.AddID()
	}
	return mc.Card, err
}

// GetCards Gets all cards
func (m *Mongo) GetCards() ([]users.Card, error) {
	// TODO: add paginations
	collection := m.Client.Database(mongoDatabase).Collection("cards")
	findOptions := options.Find()
	findOptions.SetLimit(totalRows)
	cur, err := collection.Find(context.Background(), bson.D{{}}, findOptions)
	if err != nil {
		return []users.Card{}, err
	}
	defer cur.Close(context.Background())

	cards := make([]users.Card, 0)
	for cur.Next(context.Background()) {
		var mc MongoCard
		err := cur.Decode(&mc)
		if err != nil {
			return []users.Card{}, err
		}
		mc.AddID()
		cards = append(cards, mc.Card)
	}

	return cards, nil
}

// CreateCard adds card to MongoDB
func (m *Mongo) CreateCard(card *users.Card, userId string) error {
	collection := m.Client.Database(mongoDatabase).Collection("cards")
	cardId := primitive.NewObjectID()
	mc := MongoCard{Card: *card, ID: cardId}

	_, err := collection.InsertOne(context.Background(), mc)
	if err != nil {
		return err
	}

	// Address for anonymous user
	if userId != "" {
		err = m.appendAttributeId("cards", mc.ID, userId)
		if err != nil {
			return err
		}
	}
	mc.AddID()
	*card = mc.Card
	return err
}

// GetAddress Gets an address by object Id
func (m *Mongo) GetAddress(id string) (users.Address, error) {
	addressId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return users.Address{}, err
	}

	collection := m.Client.Database(mongoDatabase).Collection("addresses")
	var ma MongoAddress
	err = collection.FindOne(context.Background(), bson.M{"_id": bson.M{"$eq": addressId}}).Decode(&ma)
	if err == nil {
		ma.AddID()
	}
	return ma.Address, err
}

// GetAddresses gets all addresses
func (m *Mongo) GetAddresses() ([]users.Address, error) {
	// TODO: add paginations
	collection := m.Client.Database(mongoDatabase).Collection("addresses")
	findOptions := options.Find()
	findOptions.SetLimit(totalRows)
	cur, err := collection.Find(context.Background(), bson.D{{}}, findOptions)
	if err != nil {
		return []users.Address{}, err
	}
	defer cur.Close(context.Background())

	addresses := make([]users.Address, 0)
	for cur.Next(context.Background()) {
		var ma MongoAddress
		err := cur.Decode(&ma)
		if err != nil {
			return []users.Address{}, err
		}
		ma.AddID()
		addresses = append(addresses, ma.Address)
	}

	return addresses, nil
}

// CreateAddress Inserts Address into MongoDB
func (m *Mongo) CreateAddress(address *users.Address, userId string) error {
	collection := m.Client.Database(mongoDatabase).Collection("addresses")
	addressId := primitive.NewObjectID()
	ma := MongoAddress{Address: *address, ID: addressId}

	_, err := collection.InsertOne(context.Background(), ma)
	if err != nil {
		return err
	}

	// Address for anonymous user
	if userId != "" {
		err = m.appendAttributeId("addresses", ma.ID, userId)
		if err != nil {
			return err
		}
	}
	ma.AddID()
	*address = ma.Address
	return err
}

// CreateAddress Inserts Address into MongoDB
func (m *Mongo) Delete(collectionName, id string) error {
	if collectionName == "customers" {
		user, err := m.GetUser(id)
		if err != nil {
			return err
		}

		ids := make([]primitive.ObjectID, 0)
		for _, address := range user.Addresses {
			addressId, err := primitive.ObjectIDFromHex(address.ID)
			if err != nil {
				return err
			}
			ids = append(ids, addressId)
		}
		collection := m.Client.Database(mongoDatabase).Collection("addresses")
		_, err = collection.DeleteMany(context.Background(), bson.M{"_id": bson.M{"$in": ids}})
		if err != nil {
			return err
		}

		ids = make([]primitive.ObjectID, 0)
		for _, card := range user.Cards {
			cardId, err := primitive.ObjectIDFromHex(card.ID)
			if err != nil {
				return err
			}
			ids = append(ids, cardId)
		}
		collection = m.Client.Database(mongoDatabase).Collection("cards")
		_, err = collection.DeleteMany(context.Background(), bson.M{"_id": bson.M{"$in": ids}})
		if err != nil {
			return err
		}
	} else {
		collectionId, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return err
		}

		collection := m.Client.Database(mongoDatabase).Collection("customers")
		_, err = collection.UpdateMany(context.Background(), bson.M{}, bson.M{"$pull": bson.M{collectionName: collectionId}})
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Mongo) Ping() error {
	err := m.Client.Ping(context.Background(), readpref.Primary())
	return err
}
