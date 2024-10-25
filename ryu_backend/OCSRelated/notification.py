# Import smtplib for the actual sending function
import smtplib
# Import the email modules we'll need    
from email.mime.text import MIMEText


def send_notification():
    # Create a text/plain message
    msg = MIMEText("FYI")

    # Specifying the from and to addresses
    fromaddr = 'xs6@rice.edu'
    toaddrs = 'sunxiaoye07@gmail.com'

    # Writing the message (this message will appear in the email)
    msg['Subject'] = 'Experiment Finished'
    msg['From'] = fromaddr
    msg['To'] = toaddrs

    # webmail Login
    f = open('/home/xs6/webmail', 'r')
    username = 'xs6'
    password = f.read()

    # Sending the mail  
    server = smtplib.SMTP('smtp.ruf.rice.edu')
    server.starttls()
    server.login(username, password)
    server.sendmail(fromaddr, toaddrs, msg.as_string())
    server.quit()


if __name__ == '__main__':
    send_notification()
