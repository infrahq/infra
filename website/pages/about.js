import Head from 'next/head'
import { useEffect, useState } from 'react'

import Layout from '../components/Layout'

const members = [
  {
    name: 'Patrick Devine',
    role: 'CTO'
  }, {
    name: 'Eva Ho',
    role: 'Engineering'
  }, {
    name: 'Michael Chiang',
    role: 'Co-Founder'
  }, {
    name: 'Elizabeth Kim',
    role: 'Engineering'
  }, {
    name: 'Matt Williams',
    role: 'Evangelism'
  }, {
    name: 'Mike Yang',
    role: 'Engineering'
  }, {
    name: 'Steven Soroka',
    role: 'Engineering'
  }, {
    name: 'Manning Fisher',
    role: 'Design'
  }, {
    name: 'Daniel Nephin',
    role: 'Engineering'
  }, {
    name: 'Bruce MacDonald',
    role: 'Engineering'
  }, {
    name: 'Jeff Morgan',
    role: 'Co-Founder'
  }
]

const values = [
  {
    name: 'Obsess about the user',
    description: 'We think deeply about how decisions affect the user. We craft an experience for the user that empowers and delights. We win when our users win.'
  }, {
    name: 'Problem first',
    description: 'We make sure we\'re solving problems that have large scale impact. We don\'t reinvent what already exists without merit. We leverage existing tools and products where we can. We take the time to think, be creative, and make sure we\'re solving the right problem.'
  }, {
    name: 'Thoughtful & considerate',
    description: 'We never treat anyone like a number and instead like a human whose opinions, feelings, and personal experience we care about. We respect and appreciate our peers both inside and outside the company.'
  }, {
    name: 'Open to being wrong',
    description: 'We understand we\'re not going to get everything right all the time, but it\'s only by being okay with making mistakes that we will be flexible enough to try different things to find the right answers. We aim to separate ourselves from our ideas, and seek first to understand others\' perspectives. We make room for people to disagree with us, and are flexible enough to move on. Disagreement is important. It takes trust and courage to be able to move forward from disagreement.'
  }, {
    name: 'Care about craftsmanship',
    description: 'We obsess over the details that make a quality product. We care about our users having a tailored experience. We are especially drawn to simplicity as a core value of our design process. We strive to be the best at what we do, but we respect that this is a life-long journey to become masters of our craft, and we must embrace an attitude of continuous learning.'
  }, {
    name: 'Challenge the status quo',
    description: 'Everything can be improved, and we must challenge our assumptions. The best answer today may not be the best answer tomorrow, and we should watch out for opportunities to grow. Sometimes that means small improvements, and sometimes that means completely new ways of thinking.'
  }, {
    name: 'Equality',
    description: 'Everyone has something valuable to contribute, and we\'re all in this together. We don\'t hand out the grunt-work. We instead innovate to reduce that type of work, and own the rest ourselves. We want you to enjoy what you do, and work on things that give you energy, not sap it. To this end, we allow any team member to choose what specific area to work on; we trust them to make the right decision for themselves.'
  }
]

export default function About () {
  const [data, setData] = useState([])

  async function updatePostings () {
    const res = await fetch('https://api.lever.co/v0/postings/infra-hq')
    const data = await res.json()
    setData(data)
  }

  useEffect(async () => {
    updatePostings()
  }, [])

  const teams = {}
  for (const d of data) {
    const team = d.categories.team
    if (!teams[team]) {
      teams[team] = []
    }

    teams[team] = [...teams[team], d]
  }

  return (
    <main className='max-w-screen-2xl mx-auto px-8 space-y-40 mb-32'>
      <Head>
        <title>Infra - About</title>
      </Head>
      <section className='flex flex-col h-screen'>
        <div className='flex flex-1 items-center self-center max-w-4xl'>
          <h1 className='text-4xl md:text-5xl lg:text-6xl xl:text-8xl text-center md:leading-extra-tight tracking-tight mb-20 font-light'>Our mission is to securely connect the world's infrastructure.</h1>
        </div>
      </section>

      {/* Intro */}
      <section className='tracking-tight max-w-4xl mx-auto my-24'>
        <h3 className='text-2xl md:text-4xl md:tracking-tight md:leading-tight'>
          Hey there! We are the people building Infra.
          <br />
          <br />
          We're a group of builders and designers who love great design and believe in creating products that are <strong>simple</strong>, <strong>cohesive</strong>, <strong>familiar</strong> and <strong>polished</strong>.
          <br />
          <br />
          In our previous lives we worked hard to bring you a great experience through <strong><a className='underline' href='https://docker.com'>Docker for Mac & Windows</a></strong>, <strong><a className='underline' href='https://datadog.com'>Datadog</a></strong>, <strong><a className='underline' href='https://dev.twitter.com'>Twitter</a></strong>, <strong><a className='underline' href='https://consul.io'>Consul</a></strong>, <strong><a className='underline' href='https://en.wikipedia.org/wiki/VMware_ESXi'>VMware ESXi</a></strong>.
          <br />
          <br />
          These products are used daily by millions. They power billions of devices, apps and servers around the world.
          <br />
          <br />
          Most importantly, they've become great businesses we're proud of.
        </h3>
      </section>

      {/* Team */}
      <section className='my-16 lg:my-32'>
        <h2 className='text-center tracking-tight text-5xl md:text-6xl mb-12'>The team behind Infra</h2>
        <img src='https://user-images.githubusercontent.com/251292/173832111-a31ab280-f615-45f8-8236-e90b1107ccac.png' />
        <div className='flex-none grid gap-4 my-10 md:gap-8 grid-cols-2 sm:grid-cols-3 md:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5'>
          {members.map(t => (
            <div className='leading-tight py-2 md:py-0 rounded-lg' key={t.name}>
              <h4 className='text-md md:text-xl md:leading-snug'>{t.name}</h4>
              <h5 className='text-md md:text-lg text-gray-400 md:leading-snug'>{t.role}</h5>
            </div>
          ))}
        </div>
      </section>

      {/* Values */}
      <section className='my-16 lg:my-24 flex flex-col lg:flex-row'>
        <h2 className='flex-1 tracking-tight text-5xl md:text-6xl mb-12 md:mb-24 mr-24'>Our&nbsp;Values</h2>
        <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 max-w-4xl gap-10 md:gap-8 lg:gap-12'>
          {values.map(v => (
            <div key={v.name}>
              <h3 className='text-xl mb-2'>{v.name}</h3>
              <p className='text-justify text-xl text-gray-400 leading-tight'>{v.description}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Work with us */}
      <section className='my-16 lg:my-32 flex flex-col lg:flex-row'>
        <h2 className='flex-none tracking-tight text-5xl md:text-6xl mb-6 md:mb-24 md:mr-24'>Work with us</h2>
        <div className='flex flex-1 flex-col'>
          {Object.keys(teams).map(t => (
            <div key={t} className='grid grid-cols-1 md:grid-cols-4 md:gap-12'>
              <h3 className='text-2xl text-gray-400 md:ml-6 md:text-right mt-7 col-span-1'>{t}</h3>
              <div className='flex-col col-span-3'>
                {teams[t].map(r => (
                  <a key={r.id} href={r.hostedUrl}>
                    <div className='flex group flex-col rounded-lg my-3 py-4'>
                      <h4 className='text-2xl group-hover:underline leading-tight'>{r.text}</h4>
                      <h5 className='text-lg text-gray-400 leading-tight'>{r.categories.location}</h5>
                    </div>
                  </a>
                ))}
              </div>
            </div>
          ))}
        </div>
      </section>
    </main>
  )
}

About.layout = function (page) {
  return (
    <Layout>
      {page}
    </Layout>
  )
}
