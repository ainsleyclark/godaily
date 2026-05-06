# Homepage Plan

In the downloads folder (~/Downloads/GoDaily/index.html) the index file that displays the design for
the initial implementation.

I want you to create the following components and modify the homepage template file to include these
template components for an initial V1 of the homepage.

## Layout

We're going to take the layout defined in `index.html` and use it as a springboard to produce the
following sections on the homepage.

- Header
  - Contains logo and nav
- Hero
	- Contains a subscribe input box, H1 & Lead
- Why Go Daily
	- 4 Feature cards, I like the title of this section `Everything a Go developer needs` etc
	- But the lead text can be improved "Heavy lifting" isn't great.
	- The text and headings can be improved, people don't care if it's a static binary, they are
	  after the Go news. Not how it's built.
- Sources
	- An up-to-date count of all of the sources.
	- At the moment, the `index.html` page displays some logos, but it would be good if we can
	  obtain logos from these providers and keep it in `views/graphics/logos` as SVGS.
- Latest Digest OR News
	- Contains the most recent digest items? Perhapps not the FULL digest, just editors pick perhaps?
	- We can simply use the existing digest items in a loop.
	- Perhaps we need to extend the store methods to pluck the latest?
- Sample 
  - Exactly how it is right now, a nice sample.
- Footer
  - Contains logo & sub text

## Components

- Header.templ - Contains the `GoDaily` text on the left and partial call to `Nav.templ`
- Nav.templ - A predefined list of nav items, right now we'll have:
	- Subscribe (goes to Hero)
	- Features
	- Sources
	- Latest News (maybe you can think of a better name, but shows most recent)
	- Sample
- Logos.templ or Sources.templ
  - That contains all of the sources
  - Will have a heading, tag, lead prop 
- Footer.templ - Contains the <footer> as defined in `index.html`

## Notes

- Use BEM for classes & SCSS naming.
- Remove all em dashes.

Come up with a plan
