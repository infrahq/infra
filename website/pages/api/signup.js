const SENDGRID_API_KEY = process.env.SENDGRID_API_KEY
const RECAPTCHA_SECRET_KEY = process.env.RECAPTCHA_SECRET_KEY
const SENDGRID_LIST_ID = process.env.SENDDGRID_LIST_ID

export default async (req, res) => {
  if (req.method !== 'POST') {
    res.status(405).json({ error: 'method must be POST' })
    return
  }

  if (!SENDGRID_API_KEY) {
    res.status(400).json({ error: 'server not configured' })
    return
  }

  if (!req.body.email || !req.body.code) {
    res.status(400).json({ error: 'invalid email or code' })
    return
  }

  const recaptchaRes = await fetch('https://www.google.com/recaptcha/api/siteverify ', {
    method: 'POST',
    body: JSON.stringify({
      secret: RECAPTCHA_SECRET_KEY,
      response: req.body.code,
      remoteip: req.ip
    })
  })

  if (!recaptchaRes.data || !recaptchaRes.data.success) {
    console.error(recaptchaRes['error-codes'])
    res.status(403)
    return
  }

  try {
    await fetch('https://api.sendgrid.com/v3/marketing/contacts', {
      method: 'PUT',
      headers: {
        Authorization: `BEARER ${SENDGRID_API_KEY}`,
        'content-type': 'application/json'
      },
      body: JSON.stringify({
        contacts: [{
          email: req.body.email
        }],
        list_ids: [
          SENDGRID_LIST_ID
        ]
      })
    })
  } catch (e) {
    console.error(e.response && e.response.data && e.response.data.errors)
    res.statusCode = 500
    return
  }

  res.status(201).json({ message: 'user added' })
}
