const { Client } = require('pg')
const { getRun } = require('util-header')

module.exports.handler = async (event, context) => {
    
    context.callbackWaitsForEmptyEventLoop = false
    let run = null
    try {
        run = await getRun(event) 
        
    } catch (error) {
        console.error('Error al obtener el run del header:: ', error)
        return { statusCode: 200 , ok : false }
    }

    const client = new Client() 
  
    try {
        await client.connect()

         const { rows } =  await client.query(
            ` 
            SELECT * 
            FROM  
                usuario
            WHERE 
                usuario.run = $1
            `, [run])

        await client.end()

        return { statusCode: 200 , data : rows[0] }

    }catch(error){
        console.error('Fallo la consulta::', error)
        if(client)
        {
            await client.end()
        }
        return { statusCode: 200 , ok : false }
    }
}