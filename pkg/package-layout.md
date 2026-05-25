# Package Layout

The architecture of the package layout has become messy and not structured very well. As the project
has grown in size, so has it's complexity. I want to simplify the codebase and conform to best
practices.

## Goal

To come up with a simple plan as a markdown file and leave it in the /docs folder.

## Problems

- The domain/news package has become bloated. Why do we have social profiles and subscribers in
  there? 
- Services don't match their domain layer. Currently there are loads of services that don't match up
  or fit anywhere. For example there is a metrics service, but no meterics domain types, which
  becomes confusing.

I want you to take a step back and write some potential folder structures of how we can improve
this.

I was thinking about having each domain layer, such as social, metrices/engagement have their own
store and service within that package (DDD), but I'm always a bit warey doing this as I don't know
where store util packages would go etc.

I basically want it easy to navigate for both me and agents.
