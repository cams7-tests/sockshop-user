db = db.getSiblingDB('users');

db.createCollection("customers");
db.createCollection("addresses");
db.createCollection("cards");

db.customers.insertMany([
    {
        "_id": ObjectId("57a98d98e4b00679b4a830af"),
        "firstName": "Eve",
        "lastName": "Berger",
        "username": "Eve_Berger",
        "password": "fec51acb3365747fc61247da5e249674cf8463c2",
        "salt": "c748112bc027878aa62812ba1ae00e40ad46d497",
        "addresses": [ObjectId("57a98d98e4b00679b4a830ad")],
        "cards": [ObjectId("57a98d98e4b00679b4a830ae")]
    },
    {
        "_id": ObjectId("57a98d98e4b00679b4a830b2"),
        "firstName": "User",
        "lastName": "Name",
        "username": "user",
        "password": "e2de7202bb2201842d041f6de201b10438369fb8",
        "salt": "6c1c6176e8b455ef37da13d953df971c249d0d8e",
        "addresses": [ObjectId("57a98d98e4b00679b4a830b0")],
        "cards": [ObjectId("57a98d98e4b00679b4a830b1")]
    },
    {
        "_id": ObjectId("57a98d98e4b00679b4a830b5"),
        "firstName": "User1",
        "lastName": "Name1",
        "username": "user1",
        "password": "8f31df4dcc25694aeb0c212118ae37bbd6e47bcd",
        "salt": "bd832b0e10c6882deabc5e8e60a37689e2b708c2",
        "addresses": [ObjectId("57a98d98e4b00679b4a830b3")],
        "cards": [ObjectId("57a98d98e4b00679b4a830b4")]
    }
]);
db.addresses.insertMany([
    {
        "_id": ObjectId("57a98d98e4b00679b4a830ad"),
        "number": "246",
        "street": "Whitelees Road",
        "city": "Glasgow",
        "postcode": "G67 3DL",
        "country": "United Kingdom"
    },
    {
        "_id": ObjectId("57a98d98e4b00679b4a830b0"),
        "number": "246",
        "street": "Whitelees Road",
        "city": "Glasgow",
        "postcode": "G67 3DL",
        "country": "United Kingdom"
    },
    {
        "_id": ObjectId("57a98d98e4b00679b4a830b3"),
        "number": "4",
        "street": "Maes-Y-Deri",
        "city": "Aberdare",
        "postcode": "CF44 6TF",
        "country": "United Kingdom"
    },
    {
        "_id": ObjectId("57a98ddce4b00679b4a830d1"),
        "number": "3",
        "street": "my road",
        "city": "London",
        "country": "UK"
    }
]);
db.cards.insertMany([
    {
        "_id": ObjectId("57a98d98e4b00679b4a830ae"),
        "longNum": "5953580604169678",
        "expires": "08/19",
        "ccv": "678"
    },
    {
        "_id": ObjectId("57a98d98e4b00679b4a830b1"),
        "longNum": "5544154011345918",
        "expires": "08/19",
        "ccv": "958"
    },
    {
        "_id": ObjectId("57a98d98e4b00679b4a830b4"),
        "longNum": "0908415193175205",
        "expires": "08/19",
        "ccv": "280"
    },
    {
        "_id": ObjectId("57a98ddce4b00679b4a830d2"),
        "longNum": "5429804235432",
        "expires": "04/16",
        "ccv": "432"
    }
]);