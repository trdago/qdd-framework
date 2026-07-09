const { Client } = require('pg')
module.exports.handler = async (event, context) => {
    

    context.callbackWaitsForEmptyEventLoop = false


    const client = new Client() 
    
    try {

        await client.connect()

       const { rows } =  await client.query(
        ` 
        select * from papeleta where region='REGIÓN DE TARAPACÁ'
        `)  

        await client.end()

        return { statusCode: 200 , papeleta : rows }
    }catch(error){
        console.error('Fallo la consulta::', error)
    return { statusCode: 200 , ok : false }

    }
}