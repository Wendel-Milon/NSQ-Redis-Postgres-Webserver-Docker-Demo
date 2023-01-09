# NSQ-Redis-Postgres-Webserver-Docker-Demo

<<<<<<< HEAD
## HowTO 
simply run *docker compose up*

**Warning** on initial startup *backend* will crash because PostgreSQL does some wierd shutdown/restart shannaigans.

to solve simply restart container.

=======
>>>>>>> fae6cc75a61a9a3dd2541f59afbcd82e1e26a081
### PostgreSQL 
 - stores the user via UserID and bycrpt encrypted Password
 - uses *db.sql* file to setup new Tables
 - preservs state via volume

### Redis
 - stores a uuid4 which should represent a session cookie
 - expires after 10 Minutes
 - preservs state via volume

### NSQ 
 - single Deamon & Lookup setup
 - no data backup

### Backend
 - Handles all the Logic.
 - Produces NSQ Messages
 - very limited logging
 - no test
 - Available Routes:
   - / -> default
   - /login 
   - /create -> Create a new User
   - /protected -> Can only be accessed via crsf-token
   - /JSON -> just some example JSON
   - /form -> deals with the Form on default page

### NSQ Consumer
 - Cosumes all Messages in topic "default"
 - There are 2 Channels on the "default" topic
 - On each channel there are two consumers
 - simulate work by sleeping 10 sec
 - do nothing but print message
  
### Makefile
 - is quite useless


## TODO
